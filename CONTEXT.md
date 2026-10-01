# cdd

A terminal UI for jumping to recently used projects. It remembers where you have been working and gets you back there in a keystroke.

## Language

**Project**:
A directory holding a git repository (a `.git` directory or file), or a directory added with `cdd add`, identified by its absolute path. It may sit at any depth. A directory that is neither is not a Project, and neither is a Forgotten one.
_Avoid_: Repo, workspace, folder, directory

**Added**:
Made a Project by hand with `cdd add`, whether or not it holds a `.git`. An Added directory is listed in the Picker even with no Visits, after the visited ones, and is Stale once the directory is gone. It is recorded in a cdd-owned file next to History; cdd never writes into the directory itself.
_Avoid_: Pinned, registered, tracked, bookmarked

**Forgotten**:
Hidden for good with `cdd forget`, git repository or not. A Forgotten Project is never listed in the Picker and a Scan does not seed it. `cdd add` brings it back.
_Avoid_: Deleted, removed, ignored, excluded, hidden

**Jump**:
Changing the shell's working directory to a chosen Project.
_Avoid_: cd, navigate, switch, open, go to

**Visit**:
A single recorded Jump to a Project, or Action run on one, or an entry seeded for a Project by a Scan. Changing directory by other means is not tracked.
_Avoid_: Access, hit, entry, usage

**Stale Visit**:
A Visit whose Project no longer qualifies at its path, whatever the cause: there is no `.git`, and it is not an Added directory that still exists. It contributes nothing to the Picker and is never pruned; it ages out of History.
_Avoid_: Orphan, dead entry, dangling, missing project

**History**:
The ordered record of Visits from which recent Projects are derived.
_Avoid_: Cache, log, database, store

**Scan**:
A one-off sweep of the directories given to it (the home directory by default) that seeds History with a Visit per Project found at any depth below them, dated from evidence found inside the Project itself. It skips Forgotten Projects. Nothing about which directories were swept is remembered.
_Avoid_: Import, index, crawl, rebuild

**Picker**:
The interactive screen that lists Projects and lets the user choose one to Jump to.
_Avoid_: Menu, list, finder, selector

**Query**:
The text typed into the Picker to filter it. Split on spaces into words: the last word matches a Project's name and every earlier word matches, in order, within its parent directory; a query with no space matches the whole shown path. Each word matches fuzzily and tolerates a typo or two by its length.
_Avoid_: Search, filter string, pattern

**Layout**:
One of the arrangements in which the Picker draws Projects. The List Layout, the only one today, draws them in one flat run.
_Avoid_: View, mode, style, theme, skin

**Action**:
A named command bound to a key in the Picker and run on the selected Project. Jumping on Enter is the built-in `jump` Action. Built-in Actions can be overridden by name, field by field, and the user can define others in `config.toml`. It may Jump once its command exits, or detach, running while the Picker stays open.
_Avoid_: Command, binding, hotkey, shortcut

**Wrapper**:
The shell function installed into the user's shell that turns a Project chosen in the Picker into a Jump.
_Avoid_: Hook, integration, plugin, shim, alias
