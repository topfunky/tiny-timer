# Version Control with Jujutsu

Guidelines for using Jujutsu (jj) for version control operations in this dotfiles project.

## Core Principles

- Use `jj` for version control operations. Background LLM Agent must not use `git`.
- **Pre-Commit Validation**: Run `make validate` and ensure it passes before any commit
- **User Confirmation**: Always obtain explicit user confirmation before executing any commit commands to VCS (use only `jj`).
- Follow conventional commit message format
- Use atomic commits with `jj commit`
- Sign commits when possible (using GPG or SSH signing)

## Version Control Basics

- **ALWAYS use `jj` (Jujutsu) for version control - NEVER use `git` commands**
- Make atomic commits with `jj commit`
- Follow [Conventional Commits](https://www.conventionalcommits.org/) format:
  - `feat:` for new features
  - `fix:` for bug fixes
  - `refactor:` for code refactoring
  - `test:` for adding or updating tests
  - `docs:` for documentation changes
  - `chore:` for maintenance tasks

## Committing Changes

**Minimize command overhead when committing:**

**ONLY use these two commands when committing:**
1. **Review changes**: Run `jj show` to see current changes
2. **Commit atomically**: Run `jj commit -m "message"` to commit

**Do NOT run:**
- `jj status` (not needed before committing)
- `jj diff` (use `jj show` instead)
- `jj log` (not needed after committing)
- Any other verification commands unless `jj commit` returns an error

**Example workflow:**
```bash
# Review what changed
jj show

# Commit with descriptive message
jj commit -m "fix: initialize database on launch to handle empty state"

# That's it! No need for jj status, jj diff, jj log, or other verification commands
```

**Important:**
- `jj commit` is atomic and will report errors if it fails
- Trust the command - if it succeeds, the commit is complete
- Only investigate further if you see an error message
- `jj status` and `jj diff` are fine for exploration/debugging, but not needed as part of the commit workflow

## Jujutsu Workflow Integration

### Commit Message Format
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

#### Types
- `feat`: new feature
- `fix`: bug fix
- `docs`: documentation changes
- `style`: formatting changes
- `refactor`: code refactoring
- `test`: adding tests
- `chore`: maintenance tasks
- `ci`: automation tasks
- `ai`: modifications to llm configuration

#### Examples
```
feat(bash): add rust development tools installer

Implement comprehensive installer for modern Rust-based CLI tools including
error handling, progress feedback, and dependency checking.

feat(jj): enhance log aliases with better filtering

- Add mine() revset for personal commits
- Improve default() revset with recent() filter
- Add statistical output to log commands

fix(setup): resolve symlink creation on existing files

Handle case where target symlinks already exist by checking and removing
stale links before creating new ones.

docs: update installation instructions

Add troubleshooting section and prerequisites clarification.
```
