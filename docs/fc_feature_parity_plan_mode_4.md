### Mode 4: History Stack Operations

| Feature         | zsh fc | shy fc | Implementation Status | Priority |
| --------------- | ------ | ------ | --------------------- | -------- |
| `-p` push stack | ✅     | ❌     | **Missing**           | MEDIUM   |
| `-P` pop stack  | ✅     | ❌     | **Missing**           | MEDIUM   |
| `-a` auto-pop   | ✅     | ❌     | **Missing**           | LOW      |

**zsh Context:**

- `fc -p` pushes current history to stack, starts new history
- Used for temporary isolated history contexts
- Primarily for shell functions/scripts

**shy Context:**

- Could enable "session switching" or "project-specific histories"
- **This is the database switching mechanism!**

**Proposed Design:**

```bash
# Push current database, start using a new one
shy fc -p ~/project/.shy_history.db

# Do work in isolated history context
command1
command2

# Pop back to original database
shy fc -P
```

**added context**

`shy` will need to keep track of the last database within the pushed database.
`-P` can look at that database to revert. `shy` should add the `.db` extension
to a file argument when pushing if the `.db` extension does not exist. `-P`
when no last db exists is a no-op with a 1 exit code.

## Design Proposals

### 4. History Stack (Database Switching)

**Core Concept:**

- SHY_HISTFILE
- only keep track of last db in the current db in a keyvalue table with `insert or replace`

**Usage Scenarios:**

**Scenario A: Project-Specific History**

```bash
# Working on project A
cd ~/projectA
shy fc -p ./.shy_history.db  # Push, start project-local history

# All commands now go to projectA/.shy_history.db
git status
make build

cd ~/projectB
shy fc -p ./.shy_history.db  # Push again, nested context

# Commands go to projectB/.shy_history.db
npm test

# Done with projectB
shy fc -P  # Pop back to projectA context

# Done with projectA
shy fc -P  # Pop back to global history
```

**Scenario B: Temporary Clean History**

```bash
# Start recording demo commands
shy fc -p /tmp/demo_history.db

# Record demo
echo "Step 1: Install"
echo "Step 2: Configure"

# Export for documentation
shy fc -W demo_commands.txt

# Return to normal history
shy fc -P
```

**Design Decisions:**

1. **Stack Persistence:**
   - **Recommendation:** databases are linked in a linked list format with a keyvalue table storing the last value in each database

1. **Auto-Pop:**
   - Support `-a` flag for function scope auto-pop?
   - Challenging in Go - would need shell integration
   - **Recommendation:** Manual pop only (Phase 1)

1. **Default Database:**
   - Bottom of stack is always default ~/.local/share/shy/history.db
   - Can't pop past the bottom
   - **Recommendation:** Yes, enforce this invariant

1. ** cli arguments histsize and savehistsize **
   - these should not anything.

---

## Redundancy Analysis

## Open Questions for User

### 2. History Stack Default Behavior

**Question:** Should history stack persist across shell sessions?

**Options:**

- **B:** In-memory only (clear on shell exit)
- **C:** Configurable via setting

**Recommendation:** Option A (persist)

**User Decision:** All user sessions are synced

---

## Appendix A: zsh fc Complete Syntax

```
fc -p [ -a ] [ filename [ histsize [ savehistsize ] ] ]
fc -P
```
