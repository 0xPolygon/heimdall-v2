package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// PreBlockerDuration tracks the time taken by the PreBlocker function.
	PreBlockerDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "pre_blocker_duration_seconds",
			Help:      "Time taken by PreBlocker function in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,
				0.90: 0.01,
				0.99: 0.001,
			},
		},
	)

	// BeginBlockerDuration tracks the time taken by the BeginBlocker function.
	BeginBlockerDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "begin_blocker_duration_seconds",
			Help:      "Time taken by BeginBlocker function in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,
				0.90: 0.01,
				0.99: 0.001,
			},
		},
	)

	// EndBlockerDuration tracks the time taken by the EndBlocker function.
	EndBlockerDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "end_blocker_duration_seconds",
			Help:      "Time taken by EndBlocker function in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,
				0.90: 0.01,
				0.99: 0.001,
			},
		},
	)

	// PrepareProposalDuration tracks the time taken by the PrepareProposal handler.
	PrepareProposalDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "prepare_proposal_duration_seconds",
			Help:      "Time taken by PrepareProposal handler in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,
				0.90: 0.01,
				0.99: 0.001,
			},
		},
	)

	// ProcessProposalDuration tracks the time taken by the ProcessProposal handler.
	ProcessProposalDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "process_proposal_duration_seconds",
			Help:      "Time taken by ProcessProposal handler in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,
				0.90: 0.01,
				0.99: 0.001,
			},
		},
	)

	// ExtendVoteDuration tracks the time taken by ExtendVote handler.
	ExtendVoteDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "extend_vote_duration_seconds",
			Help:      "Time taken by ExtendVote handler in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,
				0.90: 0.01,
				0.99: 0.001,
			},
		},
	)

	// VerifyVoteExtensionDuration tracks the time taken by VerifyVoteExtension handler.
	VerifyVoteExtensionDuration = promauto.NewSummary(
		prometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "verify_vote_extension_duration_seconds",
			Help:      "Time taken by VerifyVoteExtension handler in seconds",
			Objectives: map[float64]float64{
				0.50: 0.05,
				0.90: 0.01,
				0.99: 0.001,
			},
		},
	)

	// ExtendVoteBudgetExhaustedTotal counts ExtendVote handler turns where the
	// Zurich budget was exhausted before completion. Labels: phase = "side_tx_loop"
	// (truncated mid-side-tx iteration) or "pre_milestone" (skipped milestone
	// proposition entirely). This can be used to tune extendVoteBudget
	// against real Bor RPC tail latency.
	ExtendVoteBudgetExhaustedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "extend_vote_budget_exhausted_total",
			Help:      "Number of ExtendVote turns where the Zurich budget was exhausted before completion, labeled by phase",
		},
		[]string{"phase"},
	)

	// ExtendVoteElapsedSeconds tracks how much wall-clock time had elapsed inside
	// ExtendVote, labeled by the same phase values as ExtendVoteBudgetExhaustedTotal,
	// at the moment each phase's budget check ran (regardless of whether the budget
	// was exhausted). Pairs with that counter as a distribution: the counter says how
	// often the budget is hit, this says how close every turn runs to it, which is
	// what's needed to tune extendVoteBudget against real tail latency.
	ExtendVoteElapsedSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "extend_vote_elapsed_seconds",
			Help:      "Wall-clock time elapsed inside ExtendVote at each budget checkpoint, labeled by phase",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"phase"},
	)

	// MilestoneNoNewHeadersTotal counts ExtendVote turns that skipped milestone
	// proposition generation because Bor had no new header since the last one
	// used (ErrNoNewHeadersFound). This is the direct upstream trigger for the
	// milestone-proposition-generation retry/wait design work.
	MilestoneNoNewHeadersTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "milestone",
			Name:      "no_new_headers_total",
			Help:      "Number of ExtendVote turns that skipped milestone proposition generation because Bor had no new header",
		},
	)

	// MilestoneMajorityFoundTotal counts PreBlocker turns that found a supported
	// milestone proposition, labeled by the voting-power threshold that was met:
	// "two_thirds" (proposition committed) or "one_third" (pending, span rotation
	// deferred). This is the success-side denominator for the majority-search
	// failure counters — without it, failure counts have no rate to be a fraction of.
	MilestoneMajorityFoundTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "milestone",
			Name:      "majority_found_total",
			Help:      "Number of PreBlocker turns that found a supported milestone proposition, labeled by voting-power threshold",
		},
		[]string{"threshold"},
	)

	// VoteExtensionRejectedTotal counts VerifyVoteExtension turns that rejected a
	// peer's vote extension, labeled by the specific validation failure. A
	// rejected vote extension does not count toward the 2/3 (or 1/3) majority
	// threshold at the next height, making this a direct upstream cause of
	// milestone majority non-convergence.
	VoteExtensionRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "abci",
			Name:      "vote_extension_rejected_total",
			Help:      "Number of VerifyVoteExtension turns that rejected a peer's vote extension, labeled by reason",
		},
		[]string{"reason"},
	)

	// BorRPCCallDuration tracks the latency of individual Bor RPC calls made by
	// heimdall-v2, labeled by method. method must stay a bounded set of
	// wrapper/RPC names — never an endpoint URL, block number, tx hash, or error.
	BorRPCCallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "bor",
			Name:      "rpc_call_duration_seconds",
			Help:      "Latency of Bor RPC calls made by heimdall-v2, labeled by method",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method"},
	)
)

// RecordABCIHandlerDuration records the time taken for any ABCI handler.
func RecordABCIHandlerDuration(metric prometheus.Summary, start time.Time) {
	duration := time.Since(start)
	metric.Observe(duration.Seconds())
}

// RecordExtendVoteBudgetExhausted increments the budget-exhaustion counter
// for the given phase.
func RecordExtendVoteBudgetExhausted(phase string) {
	ExtendVoteBudgetExhaustedTotal.WithLabelValues(phase).Inc()
}

// RecordExtendVoteElapsed observes the elapsed time since start against the
// given phase's histogram.
func RecordExtendVoteElapsed(phase string, start time.Time) {
	ExtendVoteElapsedSeconds.WithLabelValues(phase).Observe(time.Since(start).Seconds())
}

// RecordMilestoneNoNewHeaders increments the counter for ExtendVote turns
// that skipped milestone proposition generation due to no new Bor header.
func RecordMilestoneNoNewHeaders() {
	MilestoneNoNewHeadersTotal.Inc()
}

// RecordMilestoneMajorityFound increments the majority-found counter for the
// given voting-power threshold ("two_thirds" or "one_third").
func RecordMilestoneMajorityFound(threshold string) {
	MilestoneMajorityFoundTotal.WithLabelValues(threshold).Inc()
}

// RecordVoteExtensionRejected increments the vote-extension-rejection counter
// for the given reason.
func RecordVoteExtensionRejected(reason string) {
	VoteExtensionRejectedTotal.WithLabelValues(reason).Inc()
}

// RecordBorRPCCallDuration observes the elapsed time since start against the
// given Bor RPC method's histogram.
func RecordBorRPCCallDuration(method string, start time.Time) {
	BorRPCCallDuration.WithLabelValues(method).Observe(time.Since(start).Seconds())
}
