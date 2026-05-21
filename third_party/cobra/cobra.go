package cobra

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type PositionalArgs func(*Command, []string) error

type Command struct {
	Use              string
	Aliases          []string
	Short            string
	Long             string
	SilenceUsage     bool
	Args             PositionalArgs
	RunE             func(*Command, []string) error
	PersistentPreRun func(*Command, []string)

	parent      *Command
	children    []*Command
	flags       *FlagSet
	pflags      *FlagSet
	parsedFlags map[string]string
}

func ExactArgs(n int) PositionalArgs {
	return func(cmd *Command, args []string) error {
		if len(args) != n {
			return fmt.Errorf("accepts %d arg(s), received %d", n, len(args))
		}
		return nil
	}
}

func (c *Command) AddCommand(children ...*Command) {
	for _, child := range children {
		child.parent = c
		c.children = append(c.children, child)
	}
}

func (c *Command) Flags() *FlagSet {
	if c.flags == nil {
		c.flags = newFlagSet()
	}
	return c.flags
}

func (c *Command) PersistentFlags() *FlagSet {
	if c.pflags == nil {
		c.pflags = newFlagSet()
	}
	return c.pflags
}

func (c *Command) Execute() error {
	return c.execute(os.Args[1:])
}

func (c *Command) execute(args []string) error {
	if len(args) > 0 && args[0] == "completion" {
		return c.runCompletion(args[1:])
	}
	cmd, rest := c.find(args)
	var err error
	rest, err = c.parsePersistent(rest)
	if err != nil {
		return err
	}
	rest, err = cmd.Flags().parse(rest)
	if err != nil {
		return err
	}
	if helpRequested(rest) {
		cmd.printHelp()
		return nil
	}
	if root := c.root(); root.PersistentPreRun != nil {
		root.PersistentPreRun(cmd, rest)
	}
	if cmd.Args != nil {
		if err := cmd.Args(cmd, rest); err != nil {
			return err
		}
	}
	if cmd.RunE == nil {
		cmd.printHelp()
		return nil
	}
	return cmd.RunE(cmd, rest)
}

func (c *Command) find(args []string) (*Command, []string) {
	current := c
	for len(args) > 0 {
		if strings.HasPrefix(args[0], "-") {
			return current, args
		}
		next := current.child(args[0])
		if next == nil {
			return current, args
		}
		current = next
		args = args[1:]
	}
	return current, args
}

func (c *Command) child(name string) *Command {
	for _, child := range c.children {
		if commandName(child.Use) == name {
			return child
		}
		for _, alias := range child.Aliases {
			if alias == name {
				return child
			}
		}
	}
	return nil
}

func (c *Command) parsePersistent(args []string) ([]string, error) {
	root := c.root()
	if root.pflags == nil {
		return args, nil
	}
	return root.pflags.parseKnown(args)
}

func (c *Command) root() *Command {
	for c.parent != nil {
		c = c.parent
	}
	return c
}

func (c *Command) CommandPath() string {
	parts := []string{}
	for cmd := c; cmd != nil; cmd = cmd.parent {
		parts = append([]string{commandName(cmd.Use)}, parts...)
	}
	return strings.Join(parts, " ")
}

func (c *Command) printHelp() {
	if c.Long != "" {
		fmt.Println(c.Long)
	} else if c.Short != "" {
		fmt.Println(c.Short)
	}
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  " + c.CommandPath() + " [flags]")
	if len(c.children) > 0 {
		fmt.Println("  " + c.CommandPath() + " [command]")
	}
	if len(c.children) > 0 {
		fmt.Println()
		fmt.Println("Available Commands:")
		for _, child := range c.children {
			fmt.Printf("  %-14s %s\n", commandName(child.Use), child.Short)
		}
	}
	fmt.Println()
	fmt.Println("Flags:")
	c.Flags().print()
	if c.root().pflags != nil {
		fmt.Println()
		fmt.Println("Global Flags:")
		c.root().pflags.print()
	}
}

func (c *Command) runCompletion(args []string) error {
	clean := args[:0]
	for _, arg := range args {
		if arg == "--no-upgrade" || arg == "--json" || arg == "--verbose" {
			continue
		}
		clean = append(clean, arg)
	}
	args = clean
	if len(args) != 1 {
		return fmt.Errorf("expected shell: bash, zsh, or fish")
	}
	names := c.commandNames()
	switch args[0] {
	case "bash":
		fmt.Printf("# bash completion for npc\ncomplete -W %q npc\n", strings.Join(names, " "))
	case "zsh":
		fmt.Printf("#compdef npc\n_arguments '1:command:(%s)'\n", strings.Join(names, " "))
	case "fish":
		for _, name := range names {
			fmt.Printf("complete -c npc -f -a %s\n", name)
		}
	default:
		return fmt.Errorf("unsupported shell %q", args[0])
	}
	return nil
}

func (c *Command) commandNames() []string {
	names := []string{}
	for _, child := range c.children {
		names = append(names, commandName(child.Use))
	}
	return names
}

type FlagSet struct {
	defs map[string]*flagDef
}

type flagDef struct {
	kind string
	ptr  any
}

func newFlagSet() *FlagSet {
	return &FlagSet{defs: map[string]*flagDef{}}
}

func (f *FlagSet) Bool(name string, value bool, usage string) {
	v := value
	f.BoolVar(&v, name, value, usage)
}

func (f *FlagSet) BoolVar(ptr *bool, name string, value bool, usage string) {
	*ptr = value
	f.defs[name] = &flagDef{kind: "bool", ptr: ptr}
}

func (f *FlagSet) StringVar(ptr *string, name, value, usage string) {
	*ptr = value
	f.defs[name] = &flagDef{kind: "string", ptr: ptr}
}

func (f *FlagSet) IntVar(ptr *int, name string, value int, usage string) {
	*ptr = value
	f.defs[name] = &flagDef{kind: "int", ptr: ptr}
}

func (f *FlagSet) GetBool(name string) (bool, error) {
	def := f.defs[name]
	if def == nil || def.kind != "bool" {
		return false, fmt.Errorf("unknown bool flag %s", name)
	}
	return *def.ptr.(*bool), nil
}

func (f *FlagSet) parse(args []string) ([]string, error) {
	rest := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			rest = append(rest, arg)
			continue
		}
		name, val, hasValue := splitFlag(arg)
		def := f.defs[name]
		if def == nil {
			rest = append(rest, arg)
			continue
		}
		if def.kind != "bool" && !hasValue {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --%s", name)
			}
			i++
			val = args[i]
		}
		if def.kind == "bool" && !hasValue {
			val = "true"
		}
		if err := setFlag(def, val); err != nil {
			return nil, fmt.Errorf("--%s: %w", name, err)
		}
	}
	return rest, nil
}

func (f *FlagSet) parseKnown(args []string) ([]string, error) {
	return f.parse(args)
}

func (f *FlagSet) print() {
	if len(f.defs) == 0 {
		fmt.Println("  -h, --help   help")
		return
	}
	fmt.Println("  -h, --help   help")
	for name := range f.defs {
		fmt.Println("      --" + name)
	}
}

func setFlag(def *flagDef, val string) error {
	switch def.kind {
	case "bool":
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		*def.ptr.(*bool) = parsed
	case "string":
		*def.ptr.(*string) = val
	case "int":
		parsed, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		*def.ptr.(*int) = parsed
	}
	return nil
}

func splitFlag(arg string) (string, string, bool) {
	arg = strings.TrimPrefix(arg, "--")
	if name, val, ok := strings.Cut(arg, "="); ok {
		return name, val, true
	}
	return arg, "", false
}

func commandName(use string) string {
	return strings.Fields(use)[0]
}

func helpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}
