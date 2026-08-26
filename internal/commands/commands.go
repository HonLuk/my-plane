// Package commands implements the Plane CLI command tree.
package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/HonLuk/my-plane/internal/api"
	"github.com/HonLuk/my-plane/internal/output"
)

// ExitError lets main distinguish a command failure from a parser that has
// already printed its own usage/error text.
type ExitError struct {
	Code    int
	Message string
	Silent  bool
}

func (e *ExitError) Error() string { return e.Message }

type runner struct {
	output *output.Renderer
}

type commonOptions struct {
	format  string
	cursor  string
	perPage int
	fields  string
	expand  string
}

var valueFlags = map[string]bool{
	"-f": true, "--format": true,
	"--cursor": true, "--per-page": true, "--fields": true, "--expand": true,
	"-p": true, "--project": true, "-i": true, "--issue": true,
	"--state": true, "--priority": true, "--assignee": true, "--order-by": true,
	"--name": true, "--identifier": true, "--description": true, "--description-html": true,
	"--label": true, "--start": true, "--end": true, "--type": true, "--content-type": true,
}

// Run dispatches one complete command. It does not construct an API client
// until a command needs network access, so every help command works offline.
func Run(args []string, renderer *output.Renderer) error {
	args = extractGlobalFormat(args)
	r := &runner{output: renderer}
	if len(args) == 0 {
		r.printRootHelp()
		return &ExitError{Code: 1, Silent: true}
	}
	if args[0] == "--help" || args[0] == "-h" {
		r.printRootHelp()
		return nil
	}

	switch args[0] {
	case "me":
		return r.runMe(args[1:])
	case "members":
		return r.runMembers(args[1:])
	case "projects":
		return r.runProjects(args[1:])
	case "issues":
		return r.runIssues(args[1:])
	case "comments":
		return r.runComments(args[1:])
	case "cycles":
		return r.runCycles(args[1:])
	case "modules":
		return r.runModules(args[1:])
	case "states":
		return r.runStates(args[1:])
	case "labels":
		return r.runLabels(args[1:])
	case "attachments":
		return r.runAttachments(args[1:])
	case "get-image":
		return r.runGetImage(args[1:])
	case "get-images":
		return r.runGetImages(args[1:])
	default:
		r.printRootHelp()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func extractGlobalFormat(args []string) []string {
	if len(args) < 2 {
		return args
	}
	if args[0] != "-f" && args[0] != "--format" && !strings.HasPrefix(args[0], "--format=") && !strings.HasPrefix(args[0], "-f=") {
		return args
	}
	if strings.Contains(args[0], "=") {
		return append(append([]string{}, args[1:]...), args[0])
	}
	return append(append([]string{}, args[2:]...), args[0], args[1])
}

func (r *runner) printRootHelp() {
	r.output.Println(`Usage: plane [--format table|json] <command> [options]

Plane.so CLI — manage projects, work items, cycles, modules, and more.

Available commands:
  me                         Show current user info
  members                    List workspace members
  projects                   Manage projects
  issues                     Manage work items (issues)
  comments                   Manage work item comments
  cycles                     Manage cycles (sprints)
  modules                    Manage modules
  states                     List workflow states in a project
  labels                     List labels in a project
  attachments                Manage work-item file attachments
  get-image                  Download a single image from a work item
  get-images                 Download all images from a work item

Set PLANE_API_KEY and PLANE_WORKSPACE environment variables before API calls.`)
}

func (r *runner) newFlagSet(name, description string) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(r.output.Out)
	set.Usage = func() {
		if description != "" {
			r.output.Println(description)
			r.output.Println()
		}
		set.PrintDefaults()
	}
	return set
}

func parseFlags(set *flag.FlagSet, args []string) (bool, error) {
	err := set.Parse(normalizeArgs(args))
	if errors.Is(err, flag.ErrHelp) {
		return true, nil
	}
	if err != nil {
		return false, &ExitError{Code: 1, Silent: true}
	}
	return false, nil
}

func flagWasSet(set *flag.FlagSet, names ...string) bool {
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		seen[name] = true
	}
	found := false
	set.Visit(func(flag *flag.Flag) {
		if seen[flag.Name] {
			found = true
		}
	})
	return found
}

func normalizeArgs(args []string) []string {
	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			positionals = append(positionals, args[index+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		if valueFlags[arg] && !strings.Contains(arg, "=") && index+1 < len(args) {
			flags = append(flags, args[index+1])
			index++
		}
	}
	return append(flags, positionals...)
}

func addCommon(set *flag.FlagSet, pagination bool) *commonOptions {
	options := &commonOptions{format: "table"}
	set.StringVar(&options.format, "format", "table", "Output format (default: table)")
	set.StringVar(&options.format, "f", "table", "Output format (default: table)")
	if pagination {
		set.StringVar(&options.cursor, "cursor", "", "Pagination cursor for next/prev page")
		set.IntVar(&options.perPage, "per-page", 0, "Number of results per page")
		set.StringVar(&options.fields, "fields", "", "Comma-separated fields to return (e.g., 'id,name,state')")
		set.StringVar(&options.expand, "expand", "", "Comma-separated relations to expand (e.g., 'assignees,state')")
	}
	return options
}

func validateFormat(value string) error {
	if value != "table" && value != "json" {
		return fmt.Errorf("invalid format %q: choose table or json", value)
	}
	return nil
}

func requirePositionals(args []string, count int, usage string) error {
	if len(args) != count {
		return fmt.Errorf("usage: %s", usage)
	}
	return nil
}

func (r *runner) client() (*api.Client, error) { return api.NewClientFromEnv() }

func (r *runner) request(client *api.Client, method, endpoint string, payload any) (any, error) {
	return client.DoJSON(context.Background(), method, endpoint, payload)
}

func endpoint(path string, params url.Values) string {
	if len(params) == 0 {
		return path
	}
	return path + "?" + params.Encode()
}

func addPagination(params url.Values, options *commonOptions) {
	if options.cursor != "" {
		params.Set("cursor", options.cursor)
	}
	if options.perPage != 0 {
		params.Set("per_page", fmt.Sprint(options.perPage))
	}
}

func validateIDs(values ...struct{ value, name string }) error {
	for _, item := range values {
		if err := api.ValidateID(item.value, item.name); err != nil {
			return err
		}
	}
	return nil
}

func escapeHTMLText(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	return strings.ReplaceAll(value, "'", "&#x27;")
}

func (r *runner) runMe(args []string) error {
	set := r.newFlagSet("me", "Usage: plane me [--format table|json]\n\nDisplay the authenticated user's ID, name, email, and display name.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	data, err := r.request(client, "GET", "/users/me/", nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) runMembers(args []string) error {
	set := r.newFlagSet("members", "Usage: plane members [options]\n\nList all members in the workspace with their name, email, role, and UUID.")
	options := addCommon(set, true)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	params := url.Values{}
	addPagination(params, options)
	data, err := r.request(client, "GET", endpoint("/workspaces/"+client.Workspace+"/members/", params), nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) runProjects(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		r.output.Println(`Usage: plane projects <list|get|create> [options]

Create, list, and view projects in the workspace.`)
		return nil
	}
	switch args[0] {
	case "list":
		return r.projectsList(args[1:])
	case "get":
		return r.projectsGet(args[1:])
	case "create":
		return r.projectsCreate(args[1:])
	default:
		return fmt.Errorf("unknown projects subcommand %q", args[0])
	}
}

func (r *runner) projectsList(args []string) error {
	set := r.newFlagSet("projects list", "Usage: plane projects list [options]\n\nShow all projects in the workspace with identifier, name, and UUID.")
	options := addCommon(set, true)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	params := url.Values{}
	addPagination(params, options)
	data, err := r.request(client, "GET", endpoint("/workspaces/"+client.Workspace+"/projects/", params), nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) projectsGet(args []string) error {
	set := r.newFlagSet("projects get", "Usage: plane projects get PROJECT_ID [options]\n\nView full details of a specific project by ID.")
	options := addCommon(set, false)
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if err := requirePositionals(set.Args(), 1, "plane projects get PROJECT_ID"); err != nil {
		return err
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	project := set.Args()[0]
	if err := validateIDs(struct{ value, name string }{project, "project ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	data, err := r.request(client, "GET", "/workspaces/"+client.Workspace+"/projects/"+project+"/", nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) projectsCreate(args []string) error {
	set := r.newFlagSet("projects create", "Usage: plane projects create --name NAME --identifier IDENTIFIER [options]\n\nCreate a new project with a name and short identifier.")
	options := addCommon(set, false)
	name := set.String("name", "", "Project display name (e.g., 'My App')")
	identifier := set.String("identifier", "", "Short key for work items (e.g., 'PROJ')")
	description := set.String("description", "", "Optional project description")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *name == "" || *identifier == "" {
		return errors.New("usage: plane projects create --name NAME --identifier IDENTIFIER")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	payload := map[string]any{"name": *name, "identifier": *identifier}
	if *description != "" {
		payload["description"] = *description
	}
	data, err := r.request(client, "POST", "/workspaces/"+client.Workspace+"/projects/", payload)
	if err != nil {
		return err
	}
	r.output.Println(r.output.Green("✓ Project created"))
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) runStates(args []string) error {
	return r.simpleProjectList("states", "workflow states", "/states/", args)
}

func (r *runner) runLabels(args []string) error {
	return r.simpleProjectList("labels", "labels", "/labels/", args)
}

func (r *runner) simpleProjectList(name, description, suffix string, args []string) error {
	set := r.newFlagSet(name, fmt.Sprintf("Usage: plane %s --project PROJECT_ID [options]\\n\\nList %s in a project.", name, description))
	options := addCommon(set, true)
	project := set.String("project", "", "Project ID (UUID)")
	set.StringVar(project, "p", "", "Project ID (UUID)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" {
		return fmt.Errorf("usage: plane %s --project PROJECT_ID", name)
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	params := url.Values{}
	addPagination(params, options)
	data, err := r.request(client, "GET", endpoint("/workspaces/"+client.Workspace+"/projects/"+*project+suffix, params), nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) runCycles(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		r.output.Println(`Usage: plane cycles <list|get|create> [options]

List, view, and create cycles (sprints) in projects.`)
		return nil
	}
	switch args[0] {
	case "list":
		return r.cyclesList(args[1:])
	case "get":
		return r.cyclesGet(args[1:])
	case "create":
		return r.cyclesCreate(args[1:])
	default:
		return fmt.Errorf("unknown cycles subcommand %q", args[0])
	}
}

func (r *runner) cyclesList(args []string) error {
	set := r.newFlagSet("cycles list", "Usage: plane cycles list --project PROJECT_ID [options]\n\nShow all cycles in a project with name, dates, and UUID.")
	options := addCommon(set, true)
	project := set.String("project", "", "Project ID (UUID)")
	set.StringVar(project, "p", "", "Project ID (UUID)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" {
		return errors.New("usage: plane cycles list --project PROJECT_ID")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	params := url.Values{}
	addPagination(params, options)
	data, err := r.request(client, "GET", endpoint("/workspaces/"+client.Workspace+"/projects/"+*project+"/cycles/", params), nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) cyclesGet(args []string) error {
	set := r.newFlagSet("cycles get", "Usage: plane cycles get --project PROJECT_ID CYCLE_ID [options]\n\nView full details of a specific cycle.")
	options := addCommon(set, false)
	project := set.String("project", "", "Project ID (UUID)")
	set.StringVar(project, "p", "", "Project ID (UUID)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || len(set.Args()) != 1 {
		return errors.New("usage: plane cycles get --project PROJECT_ID CYCLE_ID")
	}
	cycle := set.Args()[0]
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}, struct{ value, name string }{cycle, "cycle ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	data, err := r.request(client, "GET", "/workspaces/"+client.Workspace+"/projects/"+*project+"/cycles/"+cycle+"/", nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) cyclesCreate(args []string) error {
	set := r.newFlagSet("cycles create", "Usage: plane cycles create --project PROJECT_ID --name NAME [options]\n\nCreate a new sprint cycle with optional start/end dates.")
	options := addCommon(set, false)
	project := set.String("project", "", "Project ID (UUID)")
	set.StringVar(project, "p", "", "Project ID (UUID)")
	name := set.String("name", "", "Cycle name (e.g., 'Sprint 1')")
	start := set.String("start", "", "Start date (YYYY-MM-DD)")
	end := set.String("end", "", "End date (YYYY-MM-DD)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || *name == "" {
		return errors.New("usage: plane cycles create --project PROJECT_ID --name NAME")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	payload := map[string]any{"name": *name}
	if *start != "" {
		payload["start_date"] = *start
	}
	if *end != "" {
		payload["end_date"] = *end
	}
	data, err := r.request(client, "POST", "/workspaces/"+client.Workspace+"/projects/"+*project+"/cycles/", payload)
	if err != nil {
		return err
	}
	r.output.Println(r.output.Green("✓ Cycle created"))
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) runModules(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		r.output.Println(`Usage: plane modules <list|get|create> [options]

List, view, and create modules (feature groups) in projects.`)
		return nil
	}
	switch args[0] {
	case "list":
		return r.modulesList(args[1:])
	case "get":
		return r.modulesGet(args[1:])
	case "create":
		return r.modulesCreate(args[1:])
	default:
		return fmt.Errorf("unknown modules subcommand %q", args[0])
	}
}

func (r *runner) modulesList(args []string) error {
	set := r.newFlagSet("modules list", "Usage: plane modules list --project PROJECT_ID [options]\n\nShow all modules in a project.")
	options := addCommon(set, true)
	project := set.String("project", "", "Project ID (UUID)")
	set.StringVar(project, "p", "", "Project ID (UUID)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" {
		return errors.New("usage: plane modules list --project PROJECT_ID")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	params := url.Values{}
	addPagination(params, options)
	data, err := r.request(client, "GET", endpoint("/workspaces/"+client.Workspace+"/projects/"+*project+"/modules/", params), nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) modulesGet(args []string) error {
	set := r.newFlagSet("modules get", "Usage: plane modules get --project PROJECT_ID MODULE_ID [options]\n\nView full details of a specific module.")
	options := addCommon(set, false)
	project := set.String("project", "", "Project ID (UUID)")
	set.StringVar(project, "p", "", "Project ID (UUID)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || len(set.Args()) != 1 {
		return errors.New("usage: plane modules get --project PROJECT_ID MODULE_ID")
	}
	module := set.Args()[0]
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}, struct{ value, name string }{module, "module ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	data, err := r.request(client, "GET", "/workspaces/"+client.Workspace+"/projects/"+*project+"/modules/"+module+"/", nil)
	if err != nil {
		return err
	}
	r.output.Format(data, options.format, true)
	return nil
}

func (r *runner) modulesCreate(args []string) error {
	set := r.newFlagSet("modules create", "Usage: plane modules create --project PROJECT_ID --name NAME [options]\n\nCreate a new module with optional description and dates.")
	options := addCommon(set, false)
	project := set.String("project", "", "Project ID (UUID)")
	set.StringVar(project, "p", "", "Project ID (UUID)")
	name := set.String("name", "", "Module name")
	description := set.String("description", "", "Module description")
	start := set.String("start", "", "Start date (YYYY-MM-DD)")
	end := set.String("end", "", "Target date (YYYY-MM-DD)")
	help, err := parseFlags(set, args)
	if help || err != nil {
		return err
	}
	if *project == "" || *name == "" {
		return errors.New("usage: plane modules create --project PROJECT_ID --name NAME")
	}
	if err := validateFormat(options.format); err != nil {
		return err
	}
	if err := validateIDs(struct{ value, name string }{*project, "project ID"}); err != nil {
		return err
	}
	client, err := r.client()
	if err != nil {
		return err
	}
	payload := map[string]any{"name": *name}
	if *description != "" {
		payload["description"] = *description
	}
	if *start != "" {
		payload["start_date"] = *start
	}
	if *end != "" {
		payload["target_date"] = *end
	}
	data, err := r.request(client, "POST", "/workspaces/"+client.Workspace+"/projects/"+*project+"/modules/", payload)
	if err != nil {
		return err
	}
	r.output.Println(r.output.Green("✓ Module created"))
	r.output.Format(data, options.format, true)
	return nil
}
