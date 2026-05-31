-- 001_project_classification.sql
--
-- project_classification(name text) → 'project' | 'after-market'
--                                   | 'non-recoverable' | 'UNKNOWN'
--
-- Mirrors types/parentIssue.go projectClassification(). Keep the two in
-- sync — there is NO automated check that the Go and SQL implementations
-- agree.
--
-- Naming scheme of `name` (Jira project-charge string):
--
--   "<prefix>/<C><T>..."
--      C = category digit (0=Symmetry Software, 1=Symmetry Hardware, etc.)
--      T = type     digit (0=New, 1=Enhancement, 2=Maintenance,
--                          3=Customer Support / Bug Fix, 4=Technical Debt,
--                          5=Research / Non-Recoverable)
--
-- A name with no "/" represents bare "Non-Recoverable" work (legacy
-- placeholder used for issues created before the position-based naming
-- was introduced).
--
-- IMMUTABLE so the planner can fold it into index scans and inline it
-- during plan time. Safe because the function depends on its argument
-- only — no session state, no clock, no other tables.

CREATE OR REPLACE FUNCTION project_classification(name text) RETURNS text AS $$
DECLARE
    slash_pos int;
    type_char text;
BEGIN
    -- Empty / null charge → unknown (callers typically filter these out).
    IF name IS NULL OR name = '' THEN
        RETURN 'UNKNOWN';
    END IF;

    slash_pos := position('/' in name);

    -- No slash → bare "Non-Recoverable" placeholder.
    IF slash_pos = 0 THEN
        RETURN 'non-recoverable';
    END IF;

    -- Need a character at slash_pos + 2 (the type digit) — i.e. the
    -- string must be at least <prefix>/CT.
    IF slash_pos + 2 > length(name) THEN
        RETURN 'UNKNOWN';
    END IF;
    type_char := substring(name from slash_pos + 2 for 1);

    RETURN CASE type_char
        WHEN '0' THEN 'project'          -- New
        WHEN '1' THEN 'project'          -- Enhancement
        WHEN '2' THEN 'after-market'     -- Maintenance
        WHEN '3' THEN 'after-market'     -- Customer Support / Bug Fix
        WHEN '4' THEN 'after-market'     -- Technical Debt
        WHEN '5' THEN 'non-recoverable'  -- Research / Non-Recoverable
        ELSE 'UNKNOWN'
    END;
END;
$$ LANGUAGE plpgsql IMMUTABLE;
