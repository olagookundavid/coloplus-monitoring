package services

import (
	"context"
	"errors"
	"fmt"

	"coloplus-backend/internal/domain"
	apperrors "coloplus-backend/internal/errors"
	"coloplus-backend/internal/ports"
)

// UserService is administration of people and their site assignments.
type UserService struct {
	users  ports.UserRepository
	sites  ports.SiteRepository
	hasher ports.PasswordHasher
	tx     ports.TxManager
	audit  *AuditService
	clock  ports.Clock
	// accounts sends invitations; nil in the create-admin command, which
	// always sets a password.
	accounts *AccountService
	// recovery and notifier serve ResetMFA; notifier is nil in the
	// command-line tools.
	recovery ports.RecoveryCodeRepository
	notifier *NotificationService
}

func NewUserService(
	users ports.UserRepository, sites ports.SiteRepository, hasher ports.PasswordHasher,
	tx ports.TxManager, audit *AuditService, clock ports.Clock, accounts *AccountService,
	recovery ports.RecoveryCodeRepository, notifier *NotificationService,
) *UserService {
	return &UserService{
		users: users, sites: sites, hasher: hasher, tx: tx, audit: audit, clock: clock, accounts: accounts,
		recovery: recovery, notifier: notifier,
	}
}

type CreateUserInput struct {
	Email    string
	FullName string
	Role     domain.Role
	Password string
	SiteIDs  []string
}

// UserView is a user with their site assignments.
type UserView struct {
	User    domain.User
	SiteIDs []string
}

func (s *UserService) Create(ctx context.Context, actor domain.Actor, in CreateUserInput) (UserView, error) {
	if !actor.Can(domain.PermUsersManage) {
		return UserView{}, apperrors.ErrForbidden
	}
	// Without a password the person is invited by email to choose their own.
	invite := in.Password == ""
	if invite && s.accounts == nil {
		return UserView{}, invalid("a password is required")
	}
	user, problems := s.buildUser(in)
	if len(problems) > 0 {
		return UserView{}, invalidProblems(problems)
	}
	if err := s.checkSites(ctx, user.Role, in.SiteIDs); err != nil {
		return UserView{}, err
	}
	var err error
	if !invite {
		if user.PasswordHash, err = s.hasher.Hash(in.Password); err != nil {
			return UserView{}, fmt.Errorf("hashing password: %w", err)
		}
	}

	var created domain.User
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		if created, err = s.users.Create(ctx, user); err != nil {
			return err
		}
		if err := s.users.SetUserSites(ctx, created.ID, in.SiteIDs); err != nil {
			return err
		}
		if invite {
			if err := s.accounts.Invite(ctx, created); err != nil {
				return err
			}
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "user.created", EntityType: "user", EntityID: created.ID,
			Metadata: map[string]any{"role": string(created.Role), "site_count": len(in.SiteIDs), "invited": invite},
		})
	})
	if err != nil {
		return UserView{}, err
	}
	return UserView{User: created, SiteIDs: in.SiteIDs}, nil
}

func (s *UserService) buildUser(in CreateUserInput) (domain.User, []string) {
	var problems []string
	email := domain.NormalizeEmail(in.Email)
	if err := domain.ValidateEmail(email); err != nil {
		problems = append(problems, err.Error())
	}
	if in.FullName == "" {
		problems = append(problems, "the full name is required")
	}
	if !domain.ValidRole(in.Role) {
		problems = append(problems, fmt.Sprintf("%q is not a known role", in.Role))
	}
	if in.Password != "" {
		if err := domain.ValidatePassword(in.Password); err != nil {
			problems = append(problems, err.Error())
		}
	}
	return domain.User{Email: email, FullName: in.FullName, Role: in.Role, Active: true}, problems
}

// checkSites makes sure every assigned site exists. Administrators see all
// sites, so assignments for them are refused rather than silently ignored.
func (s *UserService) checkSites(ctx context.Context, role domain.Role, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if role == domain.RoleAdmin {
		return invalid("administrators can see every site and take no site assignments")
	}
	n, err := s.sites.CountExisting(ctx, ids)
	if errors.Is(err, apperrors.ErrNotFound) {
		return invalid("one or more assigned sites do not exist")
	}
	if err != nil {
		return err
	}
	if n != len(unique(ids)) {
		return invalid("one or more assigned sites do not exist")
	}
	return nil
}

func (s *UserService) List(ctx context.Context, actor domain.Actor) ([]domain.User, error) {
	if !actor.Can(domain.PermUsersManage) {
		return nil, apperrors.ErrForbidden
	}
	return s.users.List(ctx)
}

func (s *UserService) Get(ctx context.Context, actor domain.Actor, id string) (UserView, error) {
	if !actor.Can(domain.PermUsersManage) {
		return UserView{}, apperrors.ErrForbidden
	}
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	ids, err := s.users.SiteIDsForUser(ctx, id)
	if err != nil {
		return UserView{}, err
	}
	return UserView{User: user, SiteIDs: ids}, nil
}

// UpdateUserInput changes only the fields that are set.
type UpdateUserInput struct {
	FullName *string
	Role     *domain.Role
	Active   *bool
	Password *string
	SiteIDs  *[]string
}

func (s *UserService) Update(ctx context.Context, actor domain.Actor, id string, in UpdateUserInput) (UserView, error) {
	if !actor.Can(domain.PermUsersManage) {
		return UserView{}, apperrors.ErrForbidden
	}
	// Stops an administrator locking themselves out of the system.
	if id == actor.UserID && ((in.Active != nil && !*in.Active) || (in.Role != nil && *in.Role != actor.Role)) {
		return UserView{}, invalid("you cannot deactivate your own account or change your own role")
	}

	var view UserView
	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByID(ctx, id)
		if err != nil {
			return err
		}
		changed := map[string]any{}

		if in.FullName != nil {
			if *in.FullName == "" {
				return invalid("the full name is required")
			}
			user.FullName = *in.FullName
		}
		if in.Role != nil {
			if !domain.ValidRole(*in.Role) {
				return invalid("%q is not a known role", *in.Role)
			}
			user.Role = *in.Role
			changed["role"] = string(*in.Role)
		}
		if in.Active != nil {
			user.Active = *in.Active
			changed["active"] = *in.Active
		}
		if in.Password != nil {
			if err := domain.ValidatePassword(*in.Password); err != nil {
				return invalid("%s", err.Error())
			}
			if user.PasswordHash, err = s.hasher.Hash(*in.Password); err != nil {
				return fmt.Errorf("hashing password: %w", err)
			}
			changed["password_reset"] = true
		}
		if err := s.users.Update(ctx, user); err != nil {
			return err
		}

		ids, err := s.users.SiteIDsForUser(ctx, id)
		if err != nil {
			return err
		}
		if in.SiteIDs != nil {
			if err := s.checkSites(ctx, user.Role, *in.SiteIDs); err != nil {
				return err
			}
			ids = unique(*in.SiteIDs)
			if err := s.users.SetUserSites(ctx, id, ids); err != nil {
				return err
			}
			changed["site_count"] = len(ids)
		}
		view = UserView{User: user, SiteIDs: ids}

		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "user.updated", EntityType: "user", EntityID: id, Metadata: changed,
		})
	})
	return view, err
}

func unique(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

// CreateInitialAdmin creates the first administrator from the command line.
// It refuses once any administrator exists, so it cannot be used to add
// accounts later or to bypass the audit of who creates users.
func (s *UserService) CreateInitialAdmin(ctx context.Context, email, fullName, password string) (domain.User, error) {
	in := CreateUserInput{Email: email, FullName: fullName, Role: domain.RoleAdmin, Password: password}
	if password == "" {
		return domain.User{}, invalid("a password is required")
	}
	user, problems := s.buildUser(in)
	if len(problems) > 0 {
		return domain.User{}, invalidProblems(problems)
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hashing password: %w", err)
	}
	user.PasswordHash = hash

	var created domain.User
	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		existing, err := s.users.List(ctx)
		if err != nil {
			return err
		}
		for _, u := range existing {
			if u.Role == domain.RoleAdmin {
				return conflict("an administrator already exists; sign in and create users from the portal")
			}
		}
		if created, err = s.users.Create(ctx, user); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			Action: "user.initial_admin_created", EntityType: "user", EntityID: created.ID,
		})
	})
	return created, err
}

// ResendInvite sends a fresh invitation to someone who has not yet chosen a
// password. The earlier link stops working.
func (s *UserService) ResendInvite(ctx context.Context, actor domain.Actor, id string) error {
	if !actor.Can(domain.PermUsersManage) {
		return apperrors.ErrForbidden
	}
	if s.accounts == nil {
		return invalid("invitations are not available")
	}
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		user, err := s.users.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if !user.Invited() {
			return conflict("this person has already chosen a password; they can use \"forgot password\" if they are locked out")
		}
		if !user.Active {
			return conflict("this account is deactivated")
		}
		if err := s.accounts.Invite(ctx, user); err != nil {
			return err
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: actor.UserID, Action: "user.invite_resent", EntityType: "user", EntityID: id,
		})
	})
}

// ResetMFA switches off a person's two-step verification and recovery codes,
// for someone who has lost both their phone and their codes. They are signed
// out everywhere and set it up again at their next sign-in, and they are
// emailed. An administrator cannot reset their own: a stolen session must not
// be able to remove the second step.
func (s *UserService) ResetMFA(ctx context.Context, actor domain.Actor, id string) error {
	if !actor.Can(domain.PermUsersManage) {
		return apperrors.ErrForbidden
	}
	if id == actor.UserID {
		return fmt.Errorf("%w: ask another administrator to reset your two-step verification", apperrors.ErrForbidden)
	}
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	return s.resetMFA(ctx, user, actor.UserID, "api")
}

// ResetMFAByEmail is the same reset for the server console, when no
// administrator is left who can do it.
func (s *UserService) ResetMFAByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := s.users.GetByEmail(ctx, domain.NormalizeEmail(email))
	if err != nil {
		return domain.User{}, err
	}
	return user, s.resetMFA(ctx, user, "", "console")
}

func (s *UserService) resetMFA(ctx context.Context, user domain.User, byID, via string) error {
	return s.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := s.users.ResetTOTP(ctx, user.ID, s.clock.Now()); err != nil {
			return err
		}
		if err := s.recovery.DeleteAll(ctx, user.ID); err != nil {
			return err
		}
		if s.notifier != nil {
			subject, body := mfaResetEmail(user.FullName)
			if _, err := s.notifier.Enqueue(ctx, domain.Notification{
				Kind: "mfa-reset", Recipient: user.Email, Subject: subject, Body: body,
				DedupeKey: fmt.Sprintf("mfa-reset:%s:%d", user.ID, s.clock.Now().UnixNano()),
			}); err != nil {
				return err
			}
		}
		return s.audit.RecordInTx(ctx, domain.AuditEvent{
			ActorID: byID, Action: "auth.mfa_reset", EntityType: "user", EntityID: user.ID,
			Metadata: map[string]any{"via": via},
		})
	})
}
