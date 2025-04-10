# Tash Requirements

Tash is a TUI (Text User Interface) application for viewing and executing tasks from Taskfiles (see documentation at https://taskfile.dev).
The application uses the BubbleTea framework (https://github.com/charmbracelet/bubbletea/) for the terminal interface.

## Core Features

### Reading and Displaying Tasks

- Tasks are read from the command line output of the `task --list-all` command
- A model object is built from this output with the following structure:
    - Task ID (task name)
    - Task Description
    - Task Aliases (if any)
- Task information is displayed in a table view on the left side of the screen
- Command output is displayed in a viewport on the right side of the screen

### User Interface and Controls

- The application uses a full-terminal interface with a split layout:
    - Left panel: Task table showing ID, Description, and Aliases in columns
    - Right panel: Command output viewport
- Key controls:
    - `q`, `ctrl+c`, or `esc` - Exit the application
    - `ctrl+l` - Refresh the task list
    - `tab` - Switch focus between task table and output viewport
    - Arrow keys/`j`/`k` - Navigate within the focused component
    - `enter` - Execute selected task (planned functionality)

### Task Output

- Command output is appended to the viewport
- The viewport automatically scrolls to show the latest output

## Current Implementation Status

The application currently:
- Uses a full-terminal interface with the BubbleTea alt-screen mode
- Features a split layout with a task table and output viewport
- Displays tasks in a structured table format with columns for ID, Description, and Aliases
- Supports focus switching between UI components with visual indication
- Successfully parses and displays tasks from the `task --list-all` command
- Supports task refreshing with keyboard shortcuts
- Provides component-specific navigation (table rows, viewport scrolling)
- Automatically adjusts layout based on terminal dimensions
- Includes helpful key binding information at the bottom of the screen

## Future Enhancements
(To be implemented)
- Task selection and execution
- Detailed task information view
- Task filtering and search