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
  - `enter`/`e` - Execute selected task
  - `i` - Show detailed task information

### Task Output

- Command output is appended to the viewport
- The viewport automatically scrolls to show the latest output
- Text wrapping ensures proper display within the viewport width

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
- Executes selected tasks and displays their output in real-time
- Shows detailed task information in an overlay
- Provides visual distinction between application messages, command output, and error messages
- Implements proper text wrapping for displayed content

## Future Enhancements

### Task Parameters and Environment Variables
- Interactive prompt for task parameters when required
- Support for default parameter values with optional overrides
- Parameter validation based on task requirements
- Form-based interface for parameter input with field validation
- Environment variable configuration for tasks:
  - Global environment variable settings
  - Task-specific environment variable overrides
  - Environment presets that can be applied to tasks or groups
  - Import/export of environment configurations
  - Secure storage of sensitive environment variables

### Sequential and Parallel Task Execution
- Support for running multiple tasks in sequence (one after another)
- Support for running multiple tasks in parallel (simultaneously)
- Visual indicators for task execution status (pending, running, completed, failed)
- Ability to cancel running tasks

### Task History and Command Groups
- Maintain a history of executed tasks with timestamps and results
- Allow re-running of previously executed tasks
- Support for defining and saving task groups (sequences of tasks to be run together)
- Export/import of task group definitions
- Parameterized task groups with variable substitution

### Additional UI Improvements
- Task filtering and search capabilities
- Keyboard shortcut customization
- Theme customization options
- Status bar with additional context information
- Optional split view for parallel task output

### Extended Task Management
- Task dependencies visualization
- Task execution time tracking and statistics
- Integration with system notifications for long-running tasks