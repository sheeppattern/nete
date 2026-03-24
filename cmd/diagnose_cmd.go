package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sheeppattern/zk/internal/model"
	"github.com/sheeppattern/zk/internal/store"
)

// DiagnosticReport holds the full diagnosis results.
type DiagnosticReport struct {
	TotalNotes int               `json:"total_notes" yaml:"total_notes"`
	TotalLinks int               `json:"total_links" yaml:"total_links"`
	Errors     []DiagnosticItem  `json:"errors" yaml:"errors"`
	Warnings   []DiagnosticItem  `json:"warnings" yaml:"warnings"`
	Repaired   []DiagnosticItem  `json:"repaired" yaml:"repaired"`
	Summary    DiagnosticSummary `json:"summary" yaml:"summary"`
}

// DiagnosticItem represents a single diagnostic finding.
type DiagnosticItem struct {
	Severity string `json:"severity" yaml:"severity"`
	NoteID   string `json:"note_id" yaml:"note_id"`
	Message  string `json:"message" yaml:"message"`
}

// DiagnosticSummary provides an overview of the diagnosis.
type DiagnosticSummary struct {
	ErrorCount     int    `json:"error_count" yaml:"error_count"`
	WarningCount   int    `json:"warning_count" yaml:"warning_count"`
	RepairedCount  int    `json:"repaired_count" yaml:"repaired_count"`
	CorruptedCount int    `json:"corrupted_count" yaml:"corrupted_count"`
	HealthScore    string `json:"health_score" yaml:"health_score"`
}

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Diagnose storage for broken links, orphans, and invalid data",
	Long:  "Run diagnostic checks on the note store to find broken links, orphan notes, invalid relation types, invalid weights, duplicate IDs, and missing backlinks. Use --fix to auto-repair.",
	Example: `  zk diagnose --project P-XXXXXX
  zk diagnose --format md
  zk diagnose --fix --project P-XXXXXX`,
	RunE: runDiagnose,
}

func init() {
	diagnoseCmd.Flags().Bool("fix", false, "auto-repair broken links and missing backlinks")
	rootCmd.AddCommand(diagnoseCmd)
}

func runDiagnose(cmd *cobra.Command, args []string) error {
	fix, _ := cmd.Flags().GetBool("fix")
	s := store.NewStore(getStorePath(cmd))
	f := getFormatter()

	notes, noteErrors := s.ListNotesPartial(flagProject)

	// Build cross-project note map for link validation and backlink checks.
	allNotes := make(map[string]*model.Note)
	for _, n := range notes {
		allNotes[n.ID] = n
	}
	projects, _ := s.ListProjects()
	for _, p := range projects {
		if p.ID == flagProject {
			continue
		}
		pNotes, _ := s.ListNotesPartial(p.ID)
		for _, n := range pNotes {
			allNotes[n.ID] = n
		}
	}
	if flagProject != "" {
		gNotes, _ := s.ListNotesPartial("")
		for _, n := range gNotes {
			allNotes[n.ID] = n
		}
	}

	report := buildDiagnosticReport(notes, allNotes, fix, s)

	// Add corrupted file errors from partial listing.
	for _, ne := range noteErrors {
		report.Errors = append(report.Errors, DiagnosticItem{
			Severity: "error",
			NoteID:   "",
			Message:  fmt.Sprintf("corrupted file %s: %v", ne.FilePath, ne.Err),
		})
	}
	report.Summary.CorruptedCount = len(noteErrors)
	report.Summary.ErrorCount = len(report.Errors)
	if report.Summary.ErrorCount > 0 {
		report.Summary.HealthScore = "issues"
	}

	switch f.Format {
	case "json":
		return f.PrintJSON(report)
	case "yaml":
		return f.PrintYAML(report)
	case "md":
		printDiagnosticMD(report)
		return nil
	default:
		return fmt.Errorf("unsupported format: %s", f.Format)
	}
}

// hasAnyLink checks if any link to targetID exists regardless of relation type.
func hasAnyLink(links []model.Link, targetID string) bool {
	for _, l := range links {
		if l.TargetID == targetID {
			return true
		}
	}
	return false
}

func buildDiagnosticReport(notes []*model.Note, allNotes map[string]*model.Note, fix bool, s *store.Store) *DiagnosticReport {
	report := &DiagnosticReport{
		Errors:   []DiagnosticItem{},
		Warnings: []DiagnosticItem{},
		Repaired: []DiagnosticItem{},
	}

	// Build index of project note IDs for orphan detection.
	noteIDs := make(map[string]int)
	for _, n := range notes {
		noteIDs[n.ID]++
	}

	report.TotalNotes = len(notes)

	// Track which notes are link targets (incoming links).
	hasIncoming := make(map[string]bool)

	// Phase 1: Detect all issues first.
	type brokenLink struct {
		noteID   string
		index    int
		targetID string
	}
	type missingBacklink struct {
		sourceID     string
		targetID     string
		relationType string
		weight       float64
	}

	var brokenLinks []brokenLink
	var missingBacklinks []missingBacklink

	for _, n := range notes {
		for i, link := range n.Links {
			hasIncoming[link.TargetID] = true

			// Check broken links.
			if allNotes[link.TargetID] == nil {
				brokenLinks = append(brokenLinks, brokenLink{noteID: n.ID, index: i, targetID: link.TargetID})
				continue
			}

			// Check invalid relation types.
			if !model.IsValidRelationType(link.RelationType) {
				report.Warnings = append(report.Warnings, DiagnosticItem{
					Severity: "warning",
					NoteID:   n.ID,
					Message:  fmt.Sprintf("invalid relation type %q on link to %s", link.RelationType, link.TargetID),
				})
			}

			// Check invalid weights.
			if link.Weight < 0.0 || link.Weight > 1.0 {
				report.Errors = append(report.Errors, DiagnosticItem{
					Severity: "error",
					NoteID:   n.ID,
					Message:  fmt.Sprintf("invalid weight %.4f on link to %s (must be 0.0-1.0)", link.Weight, link.TargetID),
				})
			}
		}
	}

	// Check for missing backlinks (type-agnostic: any link B→A counts).
	for _, n := range notes {
		for _, link := range n.Links {
			target := allNotes[link.TargetID]
			if target == nil {
				continue
			}
			if !hasAnyLink(target.Links, n.ID) {
				missingBacklinks = append(missingBacklinks, missingBacklink{
					sourceID:     n.ID,
					targetID:     link.TargetID,
					relationType: link.RelationType,
					weight:       link.Weight,
				})
			}
		}
	}

	// Phase 2: Apply fixes or report issues.

	// Handle broken links.
	if fix {
		// Group broken links by note ID for batch removal.
		brokenByNote := make(map[string][]brokenLink)
		for _, bl := range brokenLinks {
			brokenByNote[bl.noteID] = append(brokenByNote[bl.noteID], bl)
		}
		for noteID, bls := range brokenByNote {
			n := allNotes[noteID]
			removed := make(map[int]bool)
			for _, bl := range bls {
				removed[bl.index] = true
			}
			cleaned := make([]model.Link, 0, len(n.Links)-len(bls))
			for i, l := range n.Links {
				if !removed[i] {
					cleaned = append(cleaned, l)
				}
			}
			n.Links = cleaned
			if err := s.UpdateNote(n); err != nil {
				for _, bl := range bls {
					report.Errors = append(report.Errors, DiagnosticItem{
						Severity: "error",
						NoteID:   bl.noteID,
						Message:  fmt.Sprintf("fix failed: could not remove broken link to %s: %v", bl.targetID, err),
					})
				}
			} else {
				for _, bl := range bls {
					report.Repaired = append(report.Repaired, DiagnosticItem{
						Severity: "repaired",
						NoteID:   bl.noteID,
						Message:  fmt.Sprintf("removed broken link %s→%s", bl.noteID, bl.targetID),
					})
					statusf("repaired: removed broken link %s→%s", bl.noteID, bl.targetID)
				}
			}
		}
	} else {
		for _, bl := range brokenLinks {
			report.Errors = append(report.Errors, DiagnosticItem{
				Severity: "error",
				NoteID:   bl.noteID,
				Message:  fmt.Sprintf("broken link: target note %q does not exist", bl.targetID),
			})
		}
	}

	// Handle missing backlinks (deduplicate: only add one backlink per pair).
	if fix {
		fixed := make(map[string]bool) // "targetID→sourceID" dedup key
		for _, mb := range missingBacklinks {
			key := mb.targetID + "→" + mb.sourceID
			if fixed[key] {
				continue
			}
			target := allNotes[mb.targetID]
			if hasAnyLink(target.Links, mb.sourceID) {
				continue // already fixed by a prior iteration
			}
			target.Links = append(target.Links, model.Link{
				TargetID:     mb.sourceID,
				RelationType: mb.relationType,
				Weight:       mb.weight,
			})
			if err := s.UpdateNote(target); err != nil {
				report.Errors = append(report.Errors, DiagnosticItem{
					Severity: "error",
					NoteID:   mb.targetID,
					Message:  fmt.Sprintf("fix failed: could not add backlink to %s: %v", mb.sourceID, err),
				})
			} else {
				report.Repaired = append(report.Repaired, DiagnosticItem{
					Severity: "repaired",
					NoteID:   mb.targetID,
					Message:  fmt.Sprintf("added backlink %s→%s (%s, %.2f)", mb.targetID, mb.sourceID, mb.relationType, mb.weight),
				})
				statusf("repaired: added backlink %s→%s (%s, %.2f)", mb.targetID, mb.sourceID, mb.relationType, mb.weight)
				fixed[key] = true
			}
		}
	} else {
		for _, mb := range missingBacklinks {
			report.Warnings = append(report.Warnings, DiagnosticItem{
				Severity: "warning",
				NoteID:   mb.sourceID,
				Message:  fmt.Sprintf("missing backlink: %s→%s (%s) exists but %s has no reverse link to %s", mb.sourceID, mb.targetID, mb.relationType, mb.targetID, mb.sourceID),
			})
		}
	}

	// Recount total links after fixes.
	totalLinks := 0
	for _, n := range notes {
		totalLinks += len(n.Links)
	}
	report.TotalLinks = totalLinks

	// Check for orphan notes: no outgoing links AND no incoming links.
	for _, n := range notes {
		hasOutgoing := len(n.Links) > 0
		if !hasOutgoing && !hasIncoming[n.ID] {
			report.Warnings = append(report.Warnings, DiagnosticItem{
				Severity: "warning",
				NoteID:   n.ID,
				Message:  "orphan note: no incoming or outgoing links",
			})
		}
	}

	// Check for duplicate IDs.
	for id, count := range noteIDs {
		if count > 1 {
			report.Errors = append(report.Errors, DiagnosticItem{
				Severity: "error",
				NoteID:   id,
				Message:  fmt.Sprintf("duplicate note ID found %d times", count),
			})
		}
	}

	// Build summary.
	report.Summary = DiagnosticSummary{
		ErrorCount:    len(report.Errors),
		WarningCount:  len(report.Warnings),
		RepairedCount: len(report.Repaired),
	}
	switch {
	case report.Summary.ErrorCount > 0:
		report.Summary.HealthScore = "issues"
	case report.Summary.WarningCount > 0:
		report.Summary.HealthScore = "warnings"
	default:
		report.Summary.HealthScore = "healthy"
	}

	return report
}

func printDiagnosticMD(report *DiagnosticReport) {
	var b strings.Builder

	fmt.Fprintf(&b, "# Diagnostic Report\n\n")
	fmt.Fprintf(&b, "**Total Notes**: %d\n", report.TotalNotes)
	fmt.Fprintf(&b, "**Total Links**: %d\n", report.TotalLinks)
	fmt.Fprintf(&b, "**Health Score**: %s\n", report.Summary.HealthScore)
	fmt.Fprintf(&b, "**Errors**: %d\n", report.Summary.ErrorCount)
	fmt.Fprintf(&b, "**Warnings**: %d\n", report.Summary.WarningCount)
	if report.Summary.RepairedCount > 0 {
		fmt.Fprintf(&b, "**Repaired**: %d\n", report.Summary.RepairedCount)
	}
	fmt.Fprintf(&b, "\n")

	if len(report.Errors) > 0 {
		fmt.Fprintf(&b, "## Errors\n\n")
		for _, item := range report.Errors {
			fmt.Fprintf(&b, "- **[%s]** %s\n", item.NoteID, item.Message)
		}
		fmt.Fprintf(&b, "\n")
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintf(&b, "## Warnings\n\n")
		for _, item := range report.Warnings {
			fmt.Fprintf(&b, "- **[%s]** %s\n", item.NoteID, item.Message)
		}
		fmt.Fprintf(&b, "\n")
	}

	if len(report.Repaired) > 0 {
		fmt.Fprintf(&b, "## Repaired\n\n")
		for _, item := range report.Repaired {
			fmt.Fprintf(&b, "- **[%s]** %s\n", item.NoteID, item.Message)
		}
		fmt.Fprintf(&b, "\n")
	}

	if len(report.Errors) == 0 && len(report.Warnings) == 0 {
		fmt.Fprintf(&b, "No issues found. Storage is healthy.\n")
	}

	fmt.Fprint(os.Stdout, b.String())
}
