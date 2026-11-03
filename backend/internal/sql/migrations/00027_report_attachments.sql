-- +goose Up
-- Evidence can also hang on a daily report (a photo of the generator meter, a
-- site log). Report evidence is Coloplus-only; the customer never sees it.
ALTER TABLE attachments DROP CONSTRAINT attachments_owner_type_check;
ALTER TABLE attachments ADD CONSTRAINT attachments_owner_type_check
    CHECK (owner_type IN ('ticket', 'incident', 'dispute', 'acceptance', 'report'));
