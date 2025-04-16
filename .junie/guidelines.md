# Tash Developer Guidelines

## Project Overview

Tash is a TUI (Text User Interface) application for viewing and executing tasks from Taskfiles. It provides an interactive terminal interface to work with tasks defined in [Taskfile.dev](https://taskfile.dev) format.

### Purpose and Goals

- Provide a user-friendly interface for working with Taskfiles
- Enable efficient task execution and monitoring
- Offer a visually appealing and intuitive terminal experience
- Simplify task management for developers and system administrators

## Project Architecture

### Directory Structure

```
tash/
├── cmd/
│   └── tash/           # Main application entry point
│       └── main.go     # Application initialization
├── docs/
│   └── assets/         # Documentation assets
├── examples/           # Example files
│   └── Taskfile.yml    # Sample Taskfile
├── internal/           # Internal packages
│   ├── task/           # Task management functionality
│   │   ├── process.go          # Task process handling
│   │   ├── process_windows.go  # Windows-specific process handling
│   │   ├── task.go             # Core task functionality
│   │   └── task_test.go        # Tests for task package
│   └── ui/             # User interface components
│       └── ui.go       # UI implementation
└── Taskfile.yml        # Project tasks for development
```

### Component Overview

1. **Task Package** (`internal/task/`)
   - Responsible for parsing task information from Taskfiles
   - Handles task execution and process management
   - Manages task output and error handling
   - Platform-specific process handling (Windows vs. Unix)

2. **UI Package** (`internal/ui/`)
   - Implements the terminal user interface using BubbleTea
   - Manages the split-screen layout with task list and output panels
   - Handles user input and keyboard shortcuts
   - Provides visual styling and formatting

3. **Main Application** (`cmd/tash/main.go`)
   - Initializes the application
   - Sets up the BubbleTea program
   - Handles application startup and shutdown

## Core Functionality

### Task Management

- **Task Parsing**: Tasks are read from the command line output of the `task --list-all` command
- **Task Model**: Each task contains:
  - Task ID (task name)
  - Task Description
  - Task Aliases (if any)
- **Task Execution**: Tasks are executed using the `task <taskname>` command
- **Process Management**: Task processes can be started, monitored, and cancelled

### User Interface

- **Layout**: Split-screen interface with:
  - Left panel: Task table showing ID, Description, and Aliases
  - Right panel: Command output viewport
  - Bottom: Help bar with keyboard shortcuts
- **Focus Management**: Tab-based focus switching between UI components
- **Styling**: Visual distinction between focused and unfocused components
- **Output Handling**: Real-time display of task output with proper formatting

### Key Controls

- **Navigation**:
  - `Tab` - Switch focus between task table and output viewport
  - Arrow keys/`j`/`k` - Navigate within the focused component
- **Actions**:
  - `Enter`/`e` - Execute selected task
  - `i` - Show detailed task information
  - `Ctrl+r` - Refresh the task list
  - `Ctrl+x` - Cancel a running task
  - `Ctrl+l` - Clear the output viewport
- **Application**:
  - `q`, `Ctrl+c`, or `Esc` - Exit the application

## Implementation Details

### Task Package

The task package provides the following key functionality:

1. **Task Parsing** (`ParseTaskLine`):
   - Parses task information from the output of `task --list-all`
   - Extracts task ID, description, and aliases

2. **Task Execution** (`ExecuteTask`):
   - Executes a task using the task command
   - Captures and forwards stdout and stderr
   - Handles task completion and errors

3. **Process Management**:
   - Platform-specific process handling for Unix and Windows
   - Process termination support (`StopTaskProcess`)

### UI Package

The UI package implements the terminal interface using the BubbleTea framework:

1. **Model Structure**:
   - Maintains the application state
   - Manages the task list and selected task
   - Handles focus state between components

2. **View Rendering**:
   - Renders the split-screen layout
   - Formats task information in a table
   - Displays command output with proper styling

3. **Update Logic**:
   - Processes user input and events
   - Updates the application state
   - Triggers commands based on user actions

## Development Workflow

### Building and Running

Use the Taskfile.yml in the root directory for common development tasks:

```bash
# Build the application
task build

# Run tests
task test

# Format code
task fmt

# Run linters
task lint

# Build and run the application
task run

# Clean build artifacts
task clean
```

### Testing

- Unit tests are located alongside the code they test
- Run tests with `task test` or `go test ./...`
- Add new tests for new functionality

### Adding Features

When adding new features:

1. Determine which package should contain the feature
2. Implement the core functionality in the appropriate package
3. Update the UI to expose the feature if needed
4. Add tests for the new functionality
5. Update documentation to reflect the changes

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

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests to ensure they pass (`task test`)
5. Format your code (`task fmt`)
6. Commit your changes (`git commit -m 'Add some amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## Resources

- [Taskfile.dev Documentation](https://taskfile.dev)
- [BubbleTea Documentation](https://github.com/charmbracelet/bubbletea)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)