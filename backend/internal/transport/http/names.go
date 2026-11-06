package httpapi

import (
	"reflect"
	"strings"

	"coloplus-backend/internal/dto"
	"coloplus-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// respondNamed writes a response after adding the readable names a screen
// needs next to the ids it carries. A response object says which name field
// belongs to which id with a tag, for example
//
//	SiteName string `json:"site_name,omitempty" names:"SiteID:site"`
//
// (the id field, then "site" or "user"). A name is only filled when its id is
// present, so whatever a mapper hides from a customer stays hidden. All the
// ids in one response are looked up together, one query per kind.
func (a *api) respondNamed(c *gin.Context, status int, payload any) {
	if a.directory != nil && payload != nil {
		cp := reflect.New(reflect.TypeOf(payload))
		cp.Elem().Set(reflect.ValueOf(payload))
		a.fillNames(c, cp.Elem())
		payload = cp.Elem().Interface()
	}
	c.JSON(status, payload)
}

type nameSlot struct {
	target reflect.Value // the string field to fill
	id     string
	kind   string
}

func (a *api) fillNames(c *gin.Context, v reflect.Value) {
	var slots []nameSlot
	collectNameSlots(v, &slots, 0)
	if len(slots) == 0 {
		return
	}
	// One lookup per kind: "user", "site", or an audit entity type such as
	// "daily_report" or "incident" for a record's label.
	ids := map[string][]string{}
	for _, s := range slots {
		ids[s.kind] = append(ids[s.kind], s.id)
	}
	ctx := c.Request.Context()
	found := map[string]services.Names{}
	for kind, list := range ids {
		switch kind {
		case "user":
			found[kind] = a.directory.Users(ctx, list...)
		case "site":
			found[kind] = a.directory.Sites(ctx, list...)
		default:
			found[kind] = a.directory.Labels(ctx, kind, list...)
		}
	}
	for _, s := range slots {
		if name := found[s.kind].Get(s.id); name != "" {
			s.target.SetString(name)
		}
	}
}

func collectNameSlots(v reflect.Value, out *[]nameSlot, depth int) {
	if depth > 6 {
		return
	}
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			collectNameSlots(v.Elem(), out, depth+1)
		}
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			return
		}
		for i := 0; i < v.Len(); i++ {
			collectNameSlots(v.Index(i), out, depth+1)
		}
	case reflect.Struct:
		if v.Type().PkgPath() == "time" {
			return
		}
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			fv := v.Field(i)
			if tag := f.Tag.Get("names"); tag != "" && fv.Kind() == reflect.String && fv.CanSet() {
				src, kind, _ := strings.Cut(tag, ":")
				if sf := v.FieldByName(src); sf.IsValid() && sf.Kind() == reflect.String && sf.String() != "" {
					*out = append(*out, nameSlot{target: fv, id: sf.String(), kind: kind})
				}
				continue
			}
			collectNameSlots(fv, out, depth+1)
		}
	}
}

// nameUserSites fills Sites (id and name) on each user from their SiteIDs, with
// one site lookup for the whole list.
func (a *api) nameUserSites(c *gin.Context, users []dto.UserDetailDTO) {
	var ids []string
	for _, u := range users {
		ids = append(ids, u.SiteIDs...)
	}
	names := a.directory.Sites(c.Request.Context(), ids...)
	for i := range users {
		refs := make([]dto.SiteRefDTO, 0, len(users[i].SiteIDs))
		for _, id := range users[i].SiteIDs {
			refs = append(refs, dto.SiteRefDTO{ID: id, Name: names.Get(id)})
		}
		users[i].Sites = refs
	}
}
