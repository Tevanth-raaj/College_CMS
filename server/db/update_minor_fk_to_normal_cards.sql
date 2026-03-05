-- Update Minor System to use honour_verticals instead of normal_cards
-- Run this if hod_minor_selections table already exists with FK to normal_cards

-- Drop the old foreign key constraint
ALTER TABLE hod_minor_selections 
DROP FOREIGN KEY hod_minor_selections_ibfk_3;

-- Add new foreign key constraint to honour_verticals
ALTER TABLE hod_minor_selections 
ADD CONSTRAINT hod_minor_selections_ibfk_3 
FOREIGN KEY (vertical_id) REFERENCES honour_verticals(id);

-- Verify the change
-- SHOW CREATE TABLE hod_minor_selections;
