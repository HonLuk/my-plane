package commands

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/HonLuk/my-plane/internal/markdown"
)

func (r *runner) runIssues(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		r.output.Println(`Usage: plane issues <list|get|create|update|assign|delete|search|get-short> [options]

List, create, update, assign, delete, and search work items.`)
		return nil
	}
	switch args[0] {
	case "list":
		return r.issuesList(args[1:])
	case "get":
		return r.issuesGet(args[1:])
	case "create":
		return r.issuesCreate(args[1:])
	case "update":
		return r.issuesUpdate(args[1:])
	case "assign":
		return r.issuesAssign(args[1:])
	case "delete":
		return r.issuesDelete(args[1:])
	case "search":
		return r.issuesSearch(args[1:])
	case "get-short":
		return r.issuesGetShort(args[1:])
	default:
		return fmt.Errorf("unknown issues subcommand %q", args[0])
	}
}

func addProjectAliases(set interface {
	String(string, string, string) *string
	StringVar(*string, string, string, string)
}) *string {
	project := set.String("project", "", "Project ID (UUID)")
	set.StringVar(project, "p", "", "Project ID (UUID)")
	return project
}

func (r *runner) issuesList(args []string) error {
	set := r.newFlagSet("issues list", "Usage: plane issues list --project PROJECT_ID [options]\n\nShow work items with optional filters for state, priority, or assignee.")
	options := addCommon(set, true)
	project := addProjectAliases(set)
	state := set.String("state", "", "Filter by state ID")
	priority := set.String("priority", "", "Filter by priority level")
	assignee := set.String("assignee", "", "Filter by assignee user ID")
	orderBy := set.String("order-by", "", "Sort field (e.g., '-created_at', 'priority')")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" {
		return errors.New("usage: plane issues list --project PROJECT_ID")
	}
	if err := validatePriority(*priority); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(
		struct{ value, name string }{*project, "project ID"},
		struct{ value, name string }{*state, "state ID"},
		struct{ value, name string }{*assignee, "assignee ID"},
	); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	params := url.Values{}
	if *state != "" {
		params.Set("state", *state)
	}
	if *priority != "" {
		params.Set("priority", *priority)
	}
	if *assignee != "" {
		params.Set("assignees", *assignee)
	}
	addPagination(params, options)
	if options.fields != "" {
		params.Set("fields", options.fields)
	}
	if options.expand != "" {
		params.Set("expand", options.expand)
	}
	if *orderBy != "" {
		params.Set("order_by", *orderBy)
	}
	data, err := r.request(client, "GET", endpoint("/workspaces/"+client.Workspace+"/projects/"+*project+"/work-items/", params), nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) issuesGet(args []string) error {
	set := r.newFlagSet("issues get", "Usage: plane issues get --project PROJECT_ID ISSUE_ID [options]\n\nView full details of a specific work item by UUID.")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || len(set.Args()) != 1 {
		return errors.New("usage: plane issues get --project PROJECT_ID ISSUE_ID")
	}
	issue := set.Args()[0]
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}, struct{ value, name string }{issue, "issue ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	data, err := r.request(client, "GET", "/workspaces/"+client.Workspace+"/projects/"+*project+"/work-items/"+issue+"/", nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) issuesCreate(args []string) error {
	set := r.newFlagSet("issues create", "Usage: plane issues create --project PROJECT_ID --name NAME [options]\n\nCreate a work item with title and optional fields.")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	name := set.String("name", "", "Work item title")
	description := set.String("description", "", "Plain text or simple Markdown")
	descriptionHTML := set.String("description-html", "", "Plane editor HTML")
	priority := set.String("priority", "", "Priority level")
	state := set.String("state", "", "State ID")
	assignee := set.String("assignee", "", "Assignee user ID")
	label := set.String("label", "", "Label ID")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || *name == "" {
		return errors.New("usage: plane issues create --project PROJECT_ID --name NAME")
	}
	if flagWasSet(set, "description") && flagWasSet(set, "description-html") {
		return errors.New("--description and --description-html are mutually exclusive")
	}
	if err := validatePriority(*priority); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(
		struct{ value, name string }{*project, "project ID"},
		struct{ value, name string }{*state, "state ID"},
		struct{ value, name string }{*assignee, "assignee ID"},
		struct{ value, name string }{*label, "label ID"},
	); err != nil {
		return err
	}
	descriptionValue, hasDescription, err := r.descriptionHTML(*description, *descriptionHTML)
	if err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	payload := map[string]any{"name": *name}
	if hasDescription {
		payload["description_html"] = descriptionValue
	}
	if *priority != "" {
		payload["priority"] = *priority
	}
	if *state != "" {
		payload["state"] = *state
	}
	if *assignee != "" {
		payload["assignees"] = []string{*assignee}
	}
	if *label != "" {
		payload["labels"] = []string{*label}
	}
	data, err := r.request(client, "POST", "/workspaces/"+client.Workspace+"/projects/"+*project+"/work-items/", payload)
	if err != nil {
		return err
	}
	r.output.Println(r.output.Green("✓ Work item created"))
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) issuesUpdate(args []string) error {
	set := r.newFlagSet("issues update", "Usage: plane issues update --project PROJECT_ID ISSUE_ID [options]\n\nModify title, description, priority, or state of a work item.")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	name := set.String("name", "", "New title")
	description := set.String("description", "", "New plain text or simple Markdown")
	descriptionHTML := set.String("description-html", "", "New Plane editor HTML")
	priority := set.String("priority", "", "New priority level")
	state := set.String("state", "", "New state ID")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || len(set.Args()) != 1 {
		return errors.New("usage: plane issues update --project PROJECT_ID ISSUE_ID")
	}
	if flagWasSet(set, "description") && flagWasSet(set, "description-html") {
		return errors.New("--description and --description-html are mutually exclusive")
	}
	if err := validatePriority(*priority); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	issue := set.Args()[0]
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}, struct{ value, name string }{issue, "issue ID"}, struct{ value, name string }{*state, "state ID"}); err != nil {
		return err
	}
	descriptionValue, hasDescription, err := r.descriptionHTML(*description, *descriptionHTML)
	if err != nil {
		return err
	}
	payload := map[string]any{}
	if *name != "" {
		payload["name"] = *name
	}
	if hasDescription {
		payload["description_html"] = descriptionValue
	}
	if *priority != "" {
		payload["priority"] = *priority
	}
	if *state != "" {
		payload["state"] = *state
	}
	if len(payload) == 0 {
		return errors.New("Nothing to update — pass at least one field.")
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	data, err := r.request(client, "PATCH", "/workspaces/"+client.Workspace+"/projects/"+*project+"/work-items/"+issue+"/", payload)
	if err != nil {
		return err
	}
	r.output.Println(r.output.Green("✓ Work item updated"))
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) issuesAssign(args []string) error {
	set := r.newFlagSet("issues assign", "Usage: plane issues assign --project PROJECT_ID ISSUE_ID USER_ID [USER_ID...]\n\nAssign a work item to one or more workspace members.")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || len(set.Args()) < 2 {
		return errors.New("usage: plane issues assign --project PROJECT_ID ISSUE_ID USER_ID [USER_ID...]")
	}
	issue := set.Args()[0]
	assignees := set.Args()[1:]
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}, struct{ value, name string }{issue, "issue ID"}); err != nil {
		return err
	}
	for _, assignee := range assignees {
		if err := validateIDs(struct{ value, name string }{assignee, "assignee ID"}); err != nil {
			return err
		}
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	payload := map[string]any{"assignees": assignees}
	data, err := r.request(client, "PATCH", "/workspaces/"+client.Workspace+"/projects/"+*project+"/work-items/"+issue+"/", payload)
	if err != nil {
		return err
	}
	r.output.Println(r.output.Green(fmt.Sprintf("✓ Assigned to %d member(s)", len(assignees))))
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) issuesDelete(args []string) error {
	set := r.newFlagSet("issues delete", "Usage: plane issues delete --project PROJECT_ID ISSUE_ID [options]\n\nPermanently delete a work item (irreversible).")
	options := addCommon(set, false)
	project := addProjectAliases(set)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || len(set.Args()) != 1 {
		return errors.New("usage: plane issues delete --project PROJECT_ID ISSUE_ID")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	issue := set.Args()[0]
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}, struct{ value, name string }{issue, "issue ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	if _, err := r.request(client, "DELETE", "/workspaces/"+client.Workspace+"/projects/"+*project+"/work-items/"+issue+"/", nil); err != nil {
		return err
	}
	r.output.Println(r.output.Green("✓ Deleted work item " + issue))
	return nil
}

func (r *runner) issuesSearch(args []string) error {
	set := r.newFlagSet("issues search", "Usage: plane issues search QUERY [options]\n\nSearch work items by text across all projects in the workspace.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 1, "plane issues search QUERY"); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	params := url.Values{"search": []string{set.Args()[0]}, "type": []string{"work_item"}}
	data, err := r.request(client, "GET", endpoint("/workspaces/"+client.Workspace+"/search/", params), nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) issuesGetShort(args []string) error {
	set := r.newFlagSet("issues get-short", "Usage: plane issues get-short PROJECT-SEQ [options]\n\nQuick access to a work item using its short identifier format.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 1, "plane issues get-short PROJECT-SEQ"); err != nil {
		return err
	}
	issueShort := set.Args()[0]
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{issueShort, "issue short ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	data, err := r.request(client, "GET", "/workspaces/"+client.Workspace+"/work-items/"+issueShort+"/", nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func validatePriority(value string) error {
	if value == "" {
		return nil
	}
	switch value {
	case "urgent", "high", "medium", "low", "none":
		return nil
	default:
		return fmt.Errorf("invalid priority %q: choose urgent, high, medium, low, or none", value)
	}
}

func (r *runner) descriptionHTML(description, explicitHTML string) (string, bool, error) {
	if description != "" && explicitHTML != "" {
		return "", false, errors.New("--description and --description-html are mutually exclusive")
	}
	if explicitHTML != "" {
		return explicitHTML, true, nil
	}
	if description == "" {
		return "", false, nil
	}
	for _, warning := range markdown.Warnings(description) {
		r.output.Errorln(r.output.Yellow("Warning: " + warning + "; use --description-html for full control."))
	}
	return markdown.ToHTML(description), true, nil
}
