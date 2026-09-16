# cdd

A terminal UI for jumping to recently used projects. It remembers where you have been working and gets you back there in a keystroke.

## Language

**Root**:
The top-level directory whose Kinds are searched for Projects.
_Avoid_: Base, home, workspace, projects dir

**Kind**:
A first-level directory under Root that groups Projects by purpose.
_Avoid_: Category, group, type, namespace

**Project**:
A directory exactly one level below a Kind, identified by its path relative to Root. It need not be a repository or contain anything in particular.
_Avoid_: Repo, workspace, folder, directory

**Jump**:
Changing the shell's working directory to a chosen Project.
_Avoid_: cd, navigate, switch, open, go to

**Visit**:
A single recorded Jump to a Project, or an entry seeded for a Project by a Scan. Changing directory by other means is not tracked.
_Avoid_: Access, hit, entry, usage

**Stale Visit**:
A Visit whose Project is no longer found under Root, whatever the cause. It contributes nothing to the Picker and is never pruned; it ages out of History.
_Avoid_: Orphan, dead entry, dangling, missing project

**History**:
The ordered record of Visits from which recent Projects are derived.
_Avoid_: Cache, log, database, store

**Scan**:
A one-off sweep of Root that seeds History with a Visit per Project, dated from evidence found inside the Project itself.
_Avoid_: Import, index, crawl, rebuild

**Picker**:
The interactive screen that lists Projects and lets the user choose one to Jump to.
_Avoid_: Menu, list, finder, selector

**Wrapper**:
The shell function installed into the user's shell that turns a Project chosen in the Picker into a Jump.
_Avoid_: Hook, integration, plugin, shim, alias
