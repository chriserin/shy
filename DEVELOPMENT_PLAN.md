# Iterative Development Plan

## Iteration 1: Core Data Capture

**Goal**: Establish basic history tracking pipeline

### Deliverables

- SQLite database schema for history entries
  - Tables: commands (id, command_text, timestamp, working_dir, exit_status, git_repo, git_branch)
- Basic CLI tool that can insert a command record
- Simple zsh hook integration (preexec/precmd)
- Manual testing that commands are being captured

### Success Criteria

- Commands executed in zsh are automatically stored in SQLite
- Database file created in expected location (~/.local/share/shy or similar)
- No noticeable performance impact during command execution

---

## Iteration 2: Basic Query Interface

**Goal**: Make captured data accessible

### Deliverables

- CLI subcommands for querying history
  - `shy list` - display recent commands
  - `shy search <term>` - search command text
  - `shy stats` - basic statistics (total commands, date range)
- Output formatting (table or simple list)
- Proper error handling for missing database

### Success Criteria

- Users can retrieve and search their command history
- Output is readable and useful
- Tool handles edge cases gracefully

---

## Iteration 3: Rich Metadata

**Goal**: Capture comprehensive context

### Deliverables

- Extended database schema
  - Add: duration, hostname, username, shell_pid, parent_pid
  - Add: git_branch, git_repo (if in git directory)
- Enhanced hook integration to capture new fields
- Query filtering by metadata fields
  - `shy list --dir <path>` - commands in specific directory
  - `shy list --failed` - commands with non-zero exit status
  - `shy list --since <date>` - time-based filtering

### Success Criteria

- All metadata fields populated correctly
- Queries can filter on any metadata dimension
- Git context captured when applicable

---

## Iteration 4: Privacy & Configuration

**Goal**: User control over what's tracked

### Deliverables

- Configuration file support (~/.config/shy/config.toml)
- Exclusion patterns
  - Exclude commands matching regex patterns
  - Exclude specific directories
  - Exclude commands with sensitive prefixes (passwords, tokens)
- Command to purge history
  - `shy purge --before <date>`
  - `shy purge --pattern <regex>`
- Configuration validation and defaults

### Success Criteria

- Users can configure what gets tracked
- Sensitive commands automatically excluded
- Purge operations work safely

---

## Iteration 5: Work Summarization

**Goal**: Enable understanding of work done

### Deliverables

- Summary generation features
  - `shy summary --today` - commands executed today
  - `shy summary --date <date>` - specific date
  - `shy summary --range <start> <end>` - date range
- Grouping by directory/project
- Timeline view showing work progression
- Optional: AI/LLM integration for narrative summary generation

### Success Criteria

- Users can quickly see what they worked on in a time period
- Output organized by project/directory context
- Useful for standup meetings, time tracking, or retrospectives

---

## Iteration 6: Advanced Queries & Analytics

**Goal**: Deep insights into command patterns

### Deliverables

- Frequency analysis
  - Most used commands
  - Most used commands per directory
  - Command patterns over time
- Duration analysis
  - Slowest commands
  - Average execution time by command pattern
- Export functionality
  - `shy export --format json/csv` - export filtered history
- Interactive query mode or SQL passthrough for power users

### Success Criteria

- Users can identify workflow patterns
- Performance bottlenecks visible
- Data exportable for external analysis

---

## Iteration 7: Intelligence & Suggestions

**Goal**: Proactive assistance

### Deliverables

- Command suggestion based on context
  - Suggest commands based on current directory
  - Suggest next command based on recent pattern
- Alias recommendation (frequently used long commands)
- Workflow detection (common command sequences)
- Integration with zsh completion system

### Success Criteria

- Suggestions feel helpful, not intrusive
- Suggestions improve over time with more data
- Easy to accept or dismiss suggestions

---

## Iteration 8: Multi-shell & Sync

**Goal**: Broader compatibility and data portability

### Deliverables

- Support for additional shells (bash, fish)
- Optional sync capability between machines
  - Export/import bundles
  - Optional remote sync (S3, Dropbox, etc.)
- Merge strategies for handling conflicts
- Cross-platform compatibility verification

### Success Criteria

- Works on multiple shells and platforms
- Users can consolidate history from multiple machines
- No data loss during sync operations

---

## Development Principles

### Iteration Guidelines

- Each iteration should be deployable and usable standalone
- Prioritize working features over completeness
- Get user feedback between iterations
- Keep backward compatibility in database schema (use migrations)

### Technical Considerations

- **Performance**: Benchmark hook execution time, target <10ms overhead
- **Testing**: Unit tests for core logic, integration tests for zsh hooks
- **Documentation**: Update README and man pages each iteration
- **Packaging**: Consider distribution via package managers (brew, apt)

### Dependencies to Evaluate

- SQLite driver (python: sqlite3, rust: rusqlite, go: mattn/go-sqlite3)
- CLI framework (python: click/typer, rust: clap, go: cobra)
- Configuration parsing (toml libraries)
- Optional: LLM APIs for summarization (OpenAI, Anthropic, local models)

---

## Iteration 0 (Immediate)

**Goal**: Project setup and foundation

### Deliverables

- Repository structure
- Choose implementation language
- Set up build system and dev environment
- Initial README with installation instructions
- Database schema design document
- License selection

### Success Criteria

- Development environment reproducible
- Clear contribution guidelines
- Architecture decisions documented
