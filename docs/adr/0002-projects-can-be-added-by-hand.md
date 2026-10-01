# Projects can be added by hand

ADR 0001 made a Project a directory holding a `.git`, found by Scan, and said a directory that is not a repository is never a Project. That left no way to jump to the directories people work in that are not repositories (notes, a scratch folder, a monorepo's subdirectory), and no way to drop a repository Scan keeps finding.

This supersedes 0001 in part. A Project is now a git repository **or** a directory the user added with `cdd add [dir]`, and `cdd forget [dir]` hides any Project, git or not, for good. The rest of 0001 stands: Scan is the only walk of the disk, it seeds History, and the Picker lists History without walking.

Both are recorded in one file, `projects`, in the directory that holds History (`$XDG_DATA_HOME/cdd`). Each line is `add <absolute path>` or `forget <absolute path>`, and the last line for a path wins, so `cdd add` brings back a Forgotten Project. Lines are appended and never rewritten, so concurrent writers lose nothing. The Picker lists an Added directory like any Project, ordered by its Visits, and after the visited ones when it has none. It is Stale once the directory is gone. A Forgotten Project is never listed and Scan does not seed it.

## Considered options

- **A marker file in the directory (a `.cdd`, say).** It travels with the directory, but cdd would write into the user's Projects, and `forget` could not hide a repository without touching it. cdd never writes into a Project.
- **Mark a Forgotten Project in History.** History is bounded and compacts its oldest lines away, so a Forgotten Project would come back when Scan next found it. The decision must be durable.
- **Store it in `config.toml`.** The file is the user's to edit by hand; a command that rewrites it would clobber their comments and formatting.
- **A directory is a Project if it has any of a list of marker files (`go.mod`, `package.json`).** A guess that is wrong both ways, and one more thing to configure.

## Consequences

- The `projects` file sits beside History, not in the state directory, so the two stay together when someone copies or deletes one data directory. It is not compacted; it grows by one short line per command.
- Forgetting is by exact path. A Forgotten repository that moves is a new Project.
- A Forgotten Project's History lines stay until they age out; the Picker simply does not list them.
