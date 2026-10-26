package agent

import (
	"context"
	"errors"
	"fmt"
)

// AbortCause names why an agent run's context was cancelled. Every cancellation
// site should attach one so the activity trace can report the real initiator.
type AbortCause struct {
	Code  string
	Short string
	Text  string
}

func (c *AbortCause) Error() string { return c.Text }

func cause(code, short, text string) *AbortCause {
	return &AbortCause{Code: code, Short: short, Text: text}
}

// Causef builds a cause that includes runtime-specific detail.
func Causef(code, short, format string, args ...any) *AbortCause {
	return &AbortCause{Code: code, Short: short, Text: fmt.Sprintf(format, args...)}
}

var (
	// Task-level execution context.
	AbortPausedByUser = cause("paused_by_user", "The user paused the task",
		"The user paused the task via the task-control endpoint (POST /api/tasks/{id}/control, action=pause). This Planner/Worker run was actively cancelled; the running intent returns to the frontier(open), and after the task resumes it is reclaimed and executed from the start")
	AbortPausedByOrchestrator = cause("paused_by_orchestrator", "The orchestration Agent paused the task",
		"The orchestration Agent called the pause_task tool to pause this task. This Planner/Worker run was actively cancelled; the running intent returns to the frontier(open) and is re-executed after resuming")
	AbortTaskDeleted = cause("task_deleted", "The task was deleted",
		"The task is being deleted (DELETE /api/tasks/{id}); the deletion barrier cancelled the task's running Planner, Worker, and main Agent; this run's result will no longer be used")
	AbortPausedOnReload = cause("paused_on_reload", "The backend restored the task's paused state",
		"On startup the backend restored the task's paused state from the state persisted in the database. This run was cancelled; normally there is no running Agent during the restore phase")
	AbortGoalMet = cause("goal_met", "The planner judged the task goals achieved",
		"The planner judged the task goals achieved and set the task to done, then cancelled the still-running Worker; those intents are marked stopped rather than failed")
	AbortSettleDrainTimeout = cause("settle_drain_timeout", "The wait time for task-timeout wrap-up is exhausted",
		"After the task reached timeout it waited for the running Worker to wrap up gracefully, but the 90-second drain grace was still insufficient, so a hard cancel was performed; the intent is marked exhausted, and facts and assets already written during wrap-up are kept")

	// Per-work context.
	AbortKilledByPlanner = cause("killed_by_planner", "The planner terminated this intent",
		"The planner called kill_work to actively terminate this intent, usually meaning the direction has derailed or has no further value; the intent is marked stopped and will not be auto-reclaimed")
	AbortWorkPausedByUser = cause("work_paused_by_user", "The user paused this Worker intent",
		"The user paused the running Worker. This call was cancelled and the intent became paused; the already-registered intents, facts, findings, and activity records are all kept, and it is re-executed from the start after resuming")
	AbortWorkCancelledByUser = cause("work_cancelled_by_user", "The user deleted this Worker intent",
		"The user deleted the running Worker. This call was cancelled; after the Worker exits the write region, the server handles the intent per the user's chosen deletion mode — a soft delete only marks it deleted and keeps all outputs, while a hard delete cascades removal of the intent and the downstream nodes supported only by it")
	AbortWorkFinished = cause("work_finished", "The Worker finished normally and released its context",
		"The Worker finished normally, and the engine released its context resources in detachWork. This is not a run interruption; if it appears in an interruption message, it means the cancellation and the teardown event raced")
	AbortPausedRaceGuard = cause("paused_race_guard", "Refused to start a new run while the task is paused",
		"While the task is paused, the engine refuses to issue a new execution context, to prevent a race between claim and pause from letting a Worker keep starting; the claimed intent returns to the frontier")

	// Main Agent and standalone conversation contexts.
	AbortChatStoppedByUser = cause("chat_stopped_by_user", "The user stopped this round of conversation",
		"The user clicked stop, actively aborting this round of the main Agent or conversation Agent run. The activity records already produced are kept, and you can keep sending the next message")
	AbortChatPausedWithTask = cause("chat_paused_with_task", "Pausing the task also aborted the main Agent conversation",
		"When the user paused the task, the running main Agent conversation was cancelled in sync. The activity records already produced are kept; after the task resumes this round's message is not auto-replayed")
	AbortChatTurnFinished = cause("chat_turn_finished", "This round of conversation finished normally and released its context",
		"This round of conversation finished normally, and the server is releasing this round's context resources. This is not a run interruption; if it appears in an interruption message, it means the cancellation and the teardown event raced")

	// Process-level and per-run hard backstop.
	AbortShutdown = cause("shutdown", "The backend process is shutting down",
		"The backend process received SIGINT or SIGTERM and is restarting, updating, or shutting down. All running Agents are cancelled; after a restart, any leftover running intents are reset to open and re-executed")
	AbortRunHardTimeout = cause("run_hard_timeout", "The single-run hard-timeout backstop was triggered",
		"The single run exceeded the soft wall-clock budget plus extra grace, meaning a model request or some tool did not return for a long time, so the normal round-boundary wrap-up could not run. Focus on checking the last tool call that did not return before the interruption")
)

// AbortReason resolves the named cause attached to a cancelled run context.
func AbortReason(ctx context.Context) (code, short, text string, ok bool) {
	c := context.Cause(ctx)
	if c == nil {
		return "", "", "", false
	}
	var ac *AbortCause
	if errors.As(c, &ac) {
		return ac.Code, ac.Short, ac.Text, true
	}
	switch {
	case errors.Is(c, context.DeadlineExceeded):
		return "deadline_exceeded", "The upstream context reached its deadline",
			"The upstream context reached its deadline, but the setter did not attach a named reason via WithTimeoutCause: " + c.Error(), true
	case errors.Is(c, context.Canceled):
		return "canceled_no_cause", "The canceller attached no named reason",
			"The upstream context was cancelled, but the canceller did not attach a named reason via context.WithCancelCause; please register the reason in agent/cancelcause.go and wire it into that cancellation point", true
	default:
		return "other", firstLine(c.Error(), 80), c.Error(), true
	}
}
