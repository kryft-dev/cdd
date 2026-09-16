# cdd

A terminal UI for jumping to recently used projects. It remembers where you have been working and gets you back there in a keystroke.

## Language

**Project**:
A directory the user works in and may want to return to. Identified by its path.
_Avoid_: Repo, workspace, folder, directory

**Visit**:
A single recorded occurrence of the user landing in a Project. Visits are what make a Project recent.
_Avoid_: Access, hit, entry, usage

**History**:
The ordered record of Visits from which recent Projects are derived.
_Avoid_: Cache, log, database, store

**Jump**:
Changing the shell's working directory to a chosen Project.
_Avoid_: cd, navigate, switch, open, go to

**Picker**:
The interactive screen that lists Projects and lets the user choose one to Jump to.
_Avoid_: Menu, list, finder, selector
