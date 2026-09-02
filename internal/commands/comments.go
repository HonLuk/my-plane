package commands

import (
	"bufio"
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
		r.output.Println(`Usage: plane comments <list|add|update|delete> [options]

List, add, update, and delete comments on work items.`)
		return nil
	}
	switch args[0] {
	case "list":
		return r.commentsList(args[1:])
	case "add":
		return r.commentsAdd(args[1:])
	case "update":
		return r.commentsUpdate(args[1:])
	case "delete":
		return r.commentsDelete(args[1:])
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
	set := r.newFlagSet("comments add", "Usage: plane comments add --project PROJECT_ID --issue ISSUE_ID [BODY | --body-html HTML] [options]\n\nPost a new comment on a work item. BODY supports plain text and simple Markdown; --body-html sends Plane editor HTML unchanged.")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	issue := set.String("issue", "", "Work item ID (UUID)")
	set.StringVar(issue, "i", "", "Work item ID (UUID)")
	bodyHTML := set.String("body-html", "", "Plane editor HTML (mutually exclusive with BODY)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || *issue == "" {
		return errors.New("usage: plane comments add --project PROJECT_ID --issue ISSUE_ID [BODY | --body-html HTML]")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}, struct{ value, name string }{*issue, "issue ID"}); err != nil {
		return err
	}
	contentHTML, err := r.commentContent(set.Args(), *bodyHTML, flagWasSet(set, "body-html"), "plane comments add --project PROJECT_ID --issue ISSUE_ID [BODY | --body-html HTML]")
	if err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	payload := map[string]any{"comment_html": contentHTML}
	data, err := r.request(client, "POST", commentCollectionEndpoint(client.Workspace, *project, *issue), payload)
	if err != nil {
		return err
	}
	if options.format == "json" {
		r.printJSON(data)
		return nil
	}
	r.output.Println(r.output.Green("✓ Comment added"))
	return nil
}

func (r *runner) commentsUpdate(args []string) error {
	set := r.newFlagSet("comments update", "Usage: plane comments update --project PROJECT_ID --issue ISSUE_ID COMMENT_ID [BODY | --body-html HTML] [options]\n\nUpdate a comment on a work item. BODY supports plain text and simple Markdown; --body-html sends Plane editor HTML unchanged.")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	issue := set.String("issue", "", "Work item ID (UUID)")
	set.StringVar(issue, "i", "", "Work item ID (UUID)")
	bodyHTML := set.String("body-html", "", "Plane editor HTML (mutually exclusive with BODY)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || *issue == "" || len(set.Args()) < 1 {
		return errors.New("usage: plane comments update --project PROJECT_ID --issue ISSUE_ID COMMENT_ID [BODY | --body-html HTML]")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	commentID := set.Args()[0]
	if err := validateIDs(
		struct{ value, name string }{*project, "project ID"},
		struct{ value, name string }{*issue, "issue ID"},
		struct{ value, name string }{commentID, "comment ID"},
	); err != nil {
		return err
	}
	contentHTML, err := r.commentContent(set.Args()[1:], *bodyHTML, flagWasSet(set, "body-html"), "plane comments update --project PROJECT_ID --issue ISSUE_ID COMMENT_ID [BODY | --body-html HTML]")
	if err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	data, err := r.request(client, "PATCH", commentEndpoint(client.Workspace, *project, *issue, commentID), map[string]any{"comment_html": contentHTML})
	if err != nil {
		return err
	}
	if options.format == "json" {
		r.printJSON(data)
		return nil
	}
	r.output.Println(r.output.Green("✓ Comment updated"))
	return nil
}

func (r *runner) commentsDelete(args []string) error {
	set := r.newFlagSet("comments delete", "Usage: plane comments delete --project PROJECT_ID --issue ISSUE_ID COMMENT_ID [--yes] [options]\n\nPermanently delete a comment from a work item (irreversible).")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	issue := set.String("issue", "", "Work item ID (UUID)")
	set.StringVar(issue, "i", "", "Work item ID (UUID)")
	yes := set.Bool("yes", false, "Skip the deletion confirmation prompt")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || *issue == "" || len(set.Args()) != 1 {
		return errors.New("usage: plane comments delete --project PROJECT_ID --issue ISSUE_ID COMMENT_ID [--yes]")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	commentID := set.Args()[0]
	if err := validateIDs(
		struct{ value, name string }{*project, "project ID"},
		struct{ value, name string }{*issue, "issue ID"},
		struct{ value, name string }{commentID, "comment ID"},
	); err != nil {
		return err
	}
	if !*yes {
		if err := r.confirmCommentDeletion(commentID); err != nil {
			return err
		}
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	if _, err := r.request(client, "DELETE", commentEndpoint(client.Workspace, *project, *issue, commentID), nil); err != nil {
		return err
	}
	if options.format == "json" {
		r.printJSON(map[string]any{"comment_id": commentID, "deleted": true})
		return nil
	}
	r.output.Println(r.output.Green("✓ Comment deleted"))
	return nil
}

// commentContent selects exactly one supported content source and converts
// plain text or simple Markdown before it reaches the comment API.
func (r *runner) commentContent(bodyArgs []string, explicitHTML string, htmlProvided bool, usage string) (string, error) {
	if len(bodyArgs) > 1 {
		return "", errors.New("usage: " + usage)
	}
	hasBody := len(bodyArgs) == 1
	if hasBody && htmlProvided {
		return "", errors.New("BODY and --body-html are mutually exclusive")
	}
	if !hasBody && !htmlProvided {
		return "", errors.New("usage: " + usage)
	}
	plainText := ""
	if hasBody {
		plainText = bodyArgs[0]
	}
	contentHTML, hasContent, err := r.contentHTML(plainText, explicitHTML, "BODY", "--body-html")
	if err != nil {
		return "", err
	}
	if !hasContent {
		return "", errors.New("comment content cannot be empty")
	}
	return contentHTML, nil
}

func (r *runner) confirmCommentDeletion(commentID string) error {
	r.output.Errorf("Delete comment %s? [y/N] ", commentID)
	if r.input == nil {
		r.output.Errorln()
		return errors.New("comment deletion cancelled: confirmation input is unavailable")
	}
	answer, err := bufio.NewReader(r.input).ReadString('\n')
	r.output.Errorln()
	if err != nil {
		return errors.New("comment deletion cancelled: confirmation was not completed")
	}
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return nil
	default:
		return errors.New("comment deletion cancelled")
	}
}

func commentCollectionEndpoint(workspace, projectID, issueID string) string {
	return "/workspaces/" + workspace + "/projects/" + projectID + "/work-items/" + issueID + "/comments/"
}

func commentEndpoint(workspace, projectID, issueID, commentID string) string {
	return commentCollectionEndpoint(workspace, projectID, issueID) + commentID + "/"
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
