-- reverse: modify "redirect_drafts" table
ALTER TABLE `redirect_drafts` DROP COLUMN `new_options`;
-- reverse: modify "redirects" table
ALTER TABLE `redirects` DROP COLUMN `options`;
