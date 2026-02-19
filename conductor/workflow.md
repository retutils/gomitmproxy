# Project Workflow

## Guiding Principles

1. **The Plan is the Source of Truth:** All work must be tracked in `plan.md`
2. **The Tech Stack is Deliberate:** Changes to the tech stack must be documented in `tech-stack.md` *before* implementation
3. **Test-Driven Development:** Write unit tests before implementing functionality
4. **High Code Coverage:** Aim for 98% code coverage for all modules
5. **User Experience First:** Every decision should prioritize user experience
6. **Non-Interactive & CI-Aware:** Prefer non-interactive commands. Use `CI=true` for watch-mode tools (tests, linters) to ensure single execution.

## Task Workflow

All tasks follow a strict lifecycle:

### Standard Task Workflow

1. **Select Task:** Choose the next available task from `plan.md` in sequential order

2. **Mark In Progress:** Before beginning work, edit `plan.md` and change the task from `[ ]` to `[~]`

3. **Write Failing Tests (Red Phase):**
   - Create a new test file for the feature or bug fix.
   - Write one or more unit tests that clearly define the expected behavior and acceptance criteria for the task.
   - **CRITICAL:** Run the tests and confirm that they fail as expected. This is the "Red" phase of TDD. Do not proceed until you have failing tests.

4. **Implement to Pass Tests (Green Phase):**
   - Write the minimum amount of application code necessary to make the failing tests pass.
   - Run the test suite again and confirm that all tests now pass. This is the "Green" phase.

5. **Refactor (Optional but Recommended):**
   - With the safety of passing tests, refactor the implementation code and the test code to improve clarity, remove duplication, and enhance performance without changing the external behavior.
   - Rerun tests to ensure they still pass after refactoring.

6. **Verify Coverage:** Run coverage reports using the project's chosen tools.
   - **Target:** 98% overall coverage and 100% coverage for new code.
   ```bash
   go test -coverprofile=coverage.out ./...
   go tool cover -func=coverage.out
   ```

7. **Document Deviations:** If implementation differs from tech stack:
   - **STOP** implementation
   - Update `tech-stack.md` with new design
   - Add dated note explaining the change
   - Resume implementation

8. **Commit Code Changes:**
   - **CRITICAL COMMIT CRITERIA:** Never commit without:
     - [ ] All tests passing
     - [ ] 100% coverage of new code
     - [ ] Cross-validation passing (if applicable)
   - Stage all code changes related to the task.
   - Propose a clear, concise commit message e.g, `feat(proxy): Add support for JA4 fingerprinting`.
   - Perform the commit.

9. **Attach Task Summary with Git Notes:**
   - **Step 9.1: Get Commit Hash:** Obtain the hash of the *just-completed commit* (`git log -1 --format="%H"`).
   - **Step 9.2: Draft Note Content:** Create a detailed summary for the completed task. This should include the task name, a summary of changes, a list of all created/modified files, and the core "why" for the change.
   - **Step 9.3: Attach Note:** Use the `git notes` command to attach the summary to the commit.
     ```bash
     git notes add -m "<note content>" <commit_hash>
     ```

10. **Get and Record Task Commit SHA:**
    - **Step 10.1: Update Plan:** Read `plan.md`, find the line for the completed task, update its status from `[~]` to `[x]`, and append the first 7 characters of the *just-completed commit's* commit hash.
    - **Step 10.2: Write Plan:** Write the updated content back to `plan.md`.

11. **Commit Plan Update:**
    - **Action:** Stage the modified `plan.md` file.
    - **Action:** Commit this change with a descriptive message (e.g., `conductor(plan): Mark task 'Create user model' as complete`).

### Phase Completion Verification and Checkpointing Protocol

**Trigger:** This protocol is executed immediately after a task is completed that also concludes a phase in `plan.md`.

1.  **Announce Protocol Start:** Inform the user that the phase is complete and the verification and checkpointing protocol has begun.

2.  **Ensure Test Coverage for Phase Changes:**
    -   **Step 2.1: Determine Phase Scope:** Scope is all changes since the previous phase checkpoint.
    -   **Step 2.2: List Changed Files:** `git diff --name-only <previous_checkpoint_sha> HEAD`
    -   **Step 2.3: Verify and Create Tests:** Ensure code files have corresponding tests validation phase goals.

3.  **Execute Automated Tests with Proactive Debugging:**
    -   Announce: "I will now run the automated test suite to verify the phase. **Command:** `go test ./...`"
    -   Execute tests and debug failures (max 2 attempts).

4.  **Propose a Detailed, Actionable Manual Verification Plan:** Analyze project context and plan to generate step-by-step user verification instructions.

5.  **Await Explicit User Feedback:** Pause for user confirmation before proceeding.

6.  **Create Checkpoint Commit:** `git commit --allow-empty -m "conductor(checkpoint): Checkpoint end of Phase X"`

7.  **Attach Auditable Verification Report using Git Notes:** Attach automated test results, manual steps, and user confirmation to the checkpoint commit.

8.  **Get and Record Phase Checkpoint SHA:** Update phase heading in `plan.md` with `[checkpoint: <sha>]`.

9. **Commit Plan Update:** `git commit -m "conductor(plan): Mark phase '<PHASE NAME>' as complete"`

10.  **Announce Completion:** Inform the user that the phase is complete and the checkpoint is created.

### Quality Gates

Before marking any task complete, verify:

- [ ] All tests pass
- [ ] Code coverage meets requirements (98% total, 100% new code)
- [ ] Code follows project's code style guidelines
- [ ] All public functions/methods are documented (GoDoc style)
- [ ] Type safety is enforced
- [ ] No linting or static analysis errors (`golangci-lint` or `go vet`)
- [ ] Documentation updated if needed
- [ ] No security vulnerabilities introduced

## Development Commands

### Setup
```bash
go mod tidy
go build -o go-mitmproxy cmd/go-mitmproxy/*.go
```

### Daily Development
```bash
go run cmd/go-mitmproxy/*.go
go test ./...
go fmt ./...
```

### Before Committing
```bash
make test # or go test ./...
go vet ./...
# Run linters if configured
```

## Testing Requirements

### Unit Testing
- Every module must have corresponding tests.
- Mock external dependencies where appropriate.
- Test both success and failure cases.

### Integration Testing
- Test complete proxy flows (e.g., Request -> Intercept -> Modify -> Response).
- Verify TLS handshake and certificate generation.

## Definition of Done

A task is complete when:

1. All code implemented to specification
2. Unit tests written and passing
3. Code coverage meets project requirements (98% total, 100% new code)
4. Documentation complete (GoDoc)
5. Code passes all configured linting and static analysis checks
6. Implementation notes added to `plan.md`
7. Changes committed with proper message (meeting strict criteria)
8. Git note with task summary attached to the commit

## Commit Guidelines

### Message Format
```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Examples
```bash
git commit -m "feat(proxy): Add support for JA4 fingerprinting"
git commit -m "fix(storage): Correct query building for HTTPQL"
```
