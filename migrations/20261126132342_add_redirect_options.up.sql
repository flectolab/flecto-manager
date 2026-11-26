-- modify "redirects" table
ALTER TABLE `redirects` ADD COLUMN `options` longtext NULL AFTER `status`;
-- modify "redirect_drafts" table
ALTER TABLE `redirect_drafts` ADD COLUMN `new_options` longtext NULL AFTER `new_status`;
