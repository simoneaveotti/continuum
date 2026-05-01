# ADR 002: Context Delivery (Conversational Model)

## Status

Proposed

## Context

When an agent resumes work, it needs context. But the user should not be forced to write structured data. The system must derive context from natural conversation.

## Core Principle

**User speaks → Agent summarizes → System distills → User confirms**

The user never writes structured context. The agent proposes. The system extracts. The user approves.

## Interaction Flow

### 1. User Works Naturally

User interacts with agent through conversation. No structured input required.

Example:
> "Ok, snapshot works but parsing is still broken, and we decided to stay local-first. Next step is fixing cleanPrefill."

### 2. Agent Proposes Summary

At meaningful points, the agent proposes where they are:

> "Let me summarize:
> - Snapshot is working
> - Parsing still has issues
> - Decision: local-first
> - Next: fix cleanPrefill
> 
> Update the task state?"

This is **part of conversation**, not a form.

### 3. System Distills (Hidden)

Internally maps to structured fields:

```
OBJECTIVE: ...
STATE: snapshot works, parsing broken
DECISIONS: local-first
NEXT: fix cleanPrefill
ISSUES: parser inconsistencies
```

This mapping is invisible unless requested.

### 4. User Controls

```
Apply? [y] yes  [e] edit  [n] no
```

- **y** → saves to snapshot/handoff
- **e** → modify before saving
- **n** → ignore, nothing stored

## Commands

### ctx capture

Triggers conversational distillation:
- Analyzes recent context (from agent summary)
- Produces natural-language summary
- Generates structured candidate
- Asks for confirmation (y/e/n)

### ctx context

Delivers current state to new agent:
- profile.md → who is the user
- project.md → what is the project
- snapshot.md → current task state
- handoff.md → last transition

Compact, operational, 500-1000 tokens.

### Edit Commands

```bash
ctx profile      # edit user profile
ctx project      # edit project context
ctx snapshot     # manual snapshot edit
```

Available if user wants, not required.

## Design Constraints

- NO autosave without user confirmation
- NO forced forms or structured input
- NO requirement to "use the system" actively
- NO transcript storage
- NO continuous parsing

## Distillation Triggers

Only when meaningful:
- decision taken
- task milestone reached
- problem resolved
- user requests capture
- session ending/handoff

Not continuously.

## Success Criteria

- User never feels forced to write structured context
- Agent can resume with minimal re-explanation
- Context transfer is fast and accurate
- User can always override or bypass
