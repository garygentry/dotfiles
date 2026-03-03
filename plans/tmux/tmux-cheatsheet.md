# TMUX CHEAT SHEET (Custom Config)

**Prefix:** `Ctrl-j`
**Alt Prefix:** `Ctrl-f`
Windows & panes start at **1**

---

## Sessions

```
tmux              # start
tmux attach       # reattach
Prefix + d        # detach (keep running)
Prefix + r        # reload config
```

---

## Windows

```
Prefix + c        # new window (cwd)
Shift + ← / →     # switch window
Ctrl+Shift + ←/→  # move window
Prefix + ,        # rename window
```

* Auto-renumber: ON
* Automatic rename: ON

---

## Panes

```
Prefix + v        # vertical split
Prefix + h        # horizontal split
```

### Move (no prefix)

```
Alt + ← → ↑ ↓
```

### Resize

```
Prefix + H J K L
```

---

## Close Pane

```
exit or Ctrl-d    # clean close (recommended)
Prefix + x        # force kill
```

---

## Synchronize Panes

```
Prefix + y        # toggle broadcast typing
```

---

## Copy Mode (vi)

```
Prefix + [        # enter
v                 # begin selection
h j k l           # move
y                 # copy to system clipboard
Prefix + p        # paste
```

---

## Clear

```
Prefix + L        # clear screen + history
```

---

## Mouse

* Click to select pane
* Scroll to navigate
* Drag to resize

---

## Performance

* `escape-time = 0`
* `history-limit = 10000`
* `repeat-time = 300`
* True color enabled
* vi-style keys

---

## Suggested Naming Pattern

```
1: editor
2: server
3: logs
4: git
```
