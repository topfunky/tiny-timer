# Agent Guidelines

## Version Control

- Use `jj` (Jujutsu) for version control
- Make atomic commits with `jj commit`
- Follow [Conventional Commits](https://www.conventionalcommits.org/) format:
  - `feat:` for new features
  - `fix:` for bug fixes
  - `refactor:` for code refactoring
  - `test:` for adding or updating tests
  - `docs:` for documentation changes
  - `chore:` for maintenance tasks

## Code Standards

- Write code as if a principal-level engineer wrote it
- Follow language-specific best practices and idioms
- Comments are minimal, terse, and explain **why**, not how
- Let the code speak for itself through clear naming and structure
- Only comment when the reasoning is non-obvious or requires context
- Make surgical, focused edits that address only the issues requested in the prompt

## Testing

- Write tests for all new functionality
- Update tests when modifying existing code
- Aim for meaningful coverage, not just high percentages

## Build System

- Use `make` tasks for common operations
- Check `Makefile` for available commands before creating new workflows
- Add new tasks to `Makefile` when introducing new workflows
