package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/HonLuk/my-plane/internal/output"
)

var commentHTMLTagRE = regexp.MustCompile(`<[^>]+>`)

func (r *runner) runComments(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		r.output.Println(`Usage: plane comments <list|add> [options]

List and add comments on work items.`)
		return nil
	}
	switch args[0] {
	case "list":
		return r.commentsList(args[1:])
	case "add":
		return r.commentsAdd(args[1:])
	default:
		return fmt.Errorf("unknown comments subcommand %q", args[0])
	}
}

func (r *runner) commentsList(args []string) error {
	set := r.newFlagSet("comments list", "Usage: plane comments list --project PROJECT_ID --issue ISSUE_ID [options]\n\nShow comments and optionally all activity history for a work item.")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	issue := set.String("issue", "", "Work item ID (UUID)")
	set.StringVar(issue, "i", "", "Work item ID (UUID)")
	all := set.Bool("all", false, "Show all activity (field changes), not just comments")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || *issue == "" {
		return errors.New("usage: plane comments list --project PROJECT_ID --issue ISSUE_ID")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}, struct{ value, name string }{*issue, "issue ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	suffix := "/comments/"
	if *all {
		suffix = "/activities/"
	}
	data, err := r.request(client, "GET", "/workspaces/"+client.Workspace+"/projects/"+*project+"/work-items/"+*issue+suffix, nil)
	if err != nil {
		return err
	}
	items := unwrapItems(data)
	if options.format == "json" {
		r.printJSON(items)
		return nil
	}
	if len(items) == 0 {
		if *all {
			r.output.Println(r.output.Dim("No activity found."))
		} else {
			r.output.Println(r.output.Dim("No comments found."))
		}
		return nil
	}
	for _, rawItem := range items {
		item, _ := rawItem.(map[string]any)
		actor := output.Resolve(firstValue(item, "actor_detail", "actor"))
		if actor == "—" {
			actor = "?"
		}
		created := truncateTime(output.Resolve(item["created_at"]))
		if *all {
			comment := firstValue(item, "comment", "new_value")
			field := output.Resolve(item["field"])
			if field == "comment" {
				r.output.Println(fmt.Sprintf("  %s  %s", r.output.Bold(actor), r.output.Dim(created)))
				r.output.Println("    " + output.Resolve(comment))
				r.output.Println()
			} else {
				verb := output.Resolve(item["verb"])
				if verb == "—" {
					verb = "changed"
				}
				r.output.Println(fmt.Sprintf("  %s  %s %s %s: %s → %s", r.output.Dim(created), actor, verb, field, output.Resolve(item["old_value"]), output.Resolve(item["new_value"])))
			}
			continue
		}
		comment := commentHTMLTagRE.ReplaceAllString(output.Resolve(item["comment_html"]), "")
		r.output.Println(fmt.Sprintf("  %s  %s", r.output.Bold(actor), r.output.Dim(created)))
		r.output.Println("    " + strings.TrimSpace(comment))
		r.output.Println()
	}
	return nil
}

func (r *runner) commentsAdd(args []string) error {
	set := r.newFlagSet("comments add", "Usage: plane comments add --project PROJECT_ID --issue ISSUE_ID BODY [options]\n\nPost a new comment on a work item.")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	issue := set.String("issue", "", "Work item ID (UUID)")
	set.StringVar(issue, "i", "", "Work item ID (UUID)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || *issue == "" || len(set.Args()) != 1 {
		return errors.New("usage: plane comments add --project PROJECT_ID --issue ISSUE_ID BODY")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}, struct{ value, name string }{*issue, "issue ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	payload := map[string]any{"comment_html": "<p>" + escapeHTMLText(set.Args()[0]) + "</p>"}
	data, err := r.request(client, "POST", "/workspaces/"+client.Workspace+"/projects/"+*project+"/work-items/"+*issue+"/comments/", payload)
	if err != nil {
		return err
	}
	r.output.Println(r.output.Green("✓ Comment added"))
	if data != nil && options.format == "json" {
		r.printJSON(data)
	}
	return nil
}

func unwrapItems(data any) []any {
	if object, ok := data.(map[string]any); ok {
		if results, ok := object["results"].([]any); ok {
			return results
		}
	}
	if items, ok := data.([]any); ok {
		return items
	}
	return []any{}
}

func firstValue(object map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := object[key]; ok {
			return value
		}
	}
	return nil
}

func truncateTime(value string) string {
	if len(value) > 19 {
		return value[:19]
	}
	return value
}

func (r *runner) printJSON(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		r.output.Errorln(r.output.Red(fmt.Sprintf("Could not encode JSON output: %v", err)))
		return
	}
	r.output.Println(string(encoded))
}
