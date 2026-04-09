-- modify "pages" table
ALTER TABLE `pages` MODIFY COLUMN `path` varchar(600) NULL COLLATE utf8mb4_uca1400_as_cs;
-- modify "page_drafts" table
ALTER TABLE `page_drafts` MODIFY COLUMN `new_path` varchar(600) NULL COLLATE utf8mb4_uca1400_as_cs;
-- modify "redirects" table
ALTER TABLE `redirects` MODIFY COLUMN `source` varchar(600) NULL COLLATE utf8mb4_uca1400_as_cs;
-- modify "redirect_drafts" table
ALTER TABLE `redirect_drafts` MODIFY COLUMN `new_source` varchar(600) NULL COLLATE utf8mb4_uca1400_as_cs;
