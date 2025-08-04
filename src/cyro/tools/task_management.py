"""
Task management tools for Cyro agents.

This module provides task tracking and delegation capabilities:
1. TodoWrite tool for agent task tracking (matches Claude Code functionality)
2. Task delegation tool for inter-agent communication
3. Integration with ManagerAgent routing system

No LangChain equivalent exists - custom implementation required.
"""

from typing import Any, Dict, List, Optional
from uuid import uuid4
from datetime import datetime, timedelta
from enum import Enum

from pydantic import BaseModel, Field
from pydantic_ai.toolsets import FunctionToolset

from cyro.config.settings import CyroConfig


class TaskStatus(str, Enum):
    """Task status enumeration."""
    PENDING = "pending"
    IN_PROGRESS = "in_progress" 
    COMPLETED = "completed"
    BLOCKED = "blocked"
    CANCELLED = "cancelled"


class TaskPriority(str, Enum):
    """Task priority enumeration."""
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    URGENT = "urgent"


class TodoItem(BaseModel):
    """Individual todo item model."""
    
    id: str = Field(description="Unique task identifier")
    content: str = Field(description="Task description")
    status: TaskStatus = Field(description="Current task status")
    priority: TaskPriority = Field(description="Task priority level")
    created_at: datetime = Field(default_factory=datetime.now, description="Creation timestamp")
    updated_at: datetime = Field(default_factory=datetime.now, description="Last update timestamp")
    assigned_to: Optional[str] = Field(default=None, description="Agent assigned to task")
    parent_task_id: Optional[str] = Field(default=None, description="Parent task ID for subtasks")
    tags: List[str] = Field(default_factory=list, description="Task tags")
    estimated_duration: Optional[int] = Field(default=None, description="Estimated duration in minutes")


class TodoWriteRequest(BaseModel):
    """Request model for TodoWrite operations."""
    
    todos: List[Dict[str, Any]] = Field(description="List of todo items to create/update")


class TodoWriteResult(BaseModel):
    """Result model for TodoWrite operations."""
    
    success: bool = Field(description="Whether the operation succeeded")
    message: str = Field(description="Operation result message")
    todos_created: int = Field(description="Number of todos created")
    todos_updated: int = Field(description="Number of todos updated")
    total_todos: int = Field(description="Total number of todos after operation")


class TaskDelegationRequest(BaseModel):
    """Request model for task delegation."""
    
    task_description: str = Field(description="Description of the task to delegate")
    priority: TaskPriority = Field(default=TaskPriority.MEDIUM, description="Task priority")
    requirements: List[str] = Field(default_factory=list, description="Task requirements/constraints")
    preferred_agent: Optional[str] = Field(default=None, description="Preferred agent for the task")
    deadline: Optional[datetime] = Field(default=None, description="Task deadline")
    context: Dict[str, Any] = Field(default_factory=dict, description="Additional context for the task")


class TaskDelegationResult(BaseModel):
    """Result model for task delegation."""
    
    task_id: str = Field(description="Unique task identifier")
    assigned_agent: str = Field(description="Agent assigned to the task")
    confidence_score: float = Field(description="Confidence in agent selection (0-1)")
    reasoning: str = Field(description="Explanation for agent selection")
    estimated_completion: Optional[datetime] = Field(default=None, description="Estimated completion time")


class TaskQuery(BaseModel):
    """Query model for task filtering."""
    
    status: Optional[TaskStatus] = Field(default=None, description="Filter by status")
    priority: Optional[TaskPriority] = Field(default=None, description="Filter by priority")
    assigned_to: Optional[str] = Field(default=None, description="Filter by assigned agent")
    tags: List[str] = Field(default_factory=list, description="Filter by tags")
    limit: int = Field(default=50, description="Maximum number of results")


class TaskQueryResult(BaseModel):
    """Result model for task queries."""
    
    tasks: List[TodoItem] = Field(description="Matching tasks")
    total_count: int = Field(description="Total number of matching tasks")
    filtered_count: int = Field(description="Number of tasks returned")


class TaskManager:
    """Task management system for tracking and delegating tasks."""
    
    def __init__(self, config: Optional[CyroConfig] = None):
        """Initialize task manager.
        
        Args:
            config: Cyro configuration
        """
        self.config = config or CyroConfig()
        self._tasks: Dict[str, TodoItem] = {}
        self._session_todos: List[TodoItem] = []  # In-memory session todos
    
    def _generate_task_id(self) -> str:
        """Generate a unique task ID."""
        return str(uuid4())[:8]
    
    def _parse_todo_dict(self, todo_dict: Dict[str, Any]) -> TodoItem:
        """Parse a todo dictionary into a TodoItem."""
        # Handle both string and enum values for status/priority
        status = todo_dict.get("status", TaskStatus.PENDING)
        if isinstance(status, str):
            status = TaskStatus(status)
        
        priority = todo_dict.get("priority", TaskPriority.MEDIUM)
        if isinstance(priority, str):
            priority = TaskPriority(priority)
        
        return TodoItem(
            id=todo_dict.get("id", self._generate_task_id()),
            content=todo_dict["content"],
            status=status,
            priority=priority,
            assigned_to=todo_dict.get("assigned_to"),
            parent_task_id=todo_dict.get("parent_task_id"),
            tags=todo_dict.get("tags", []),
            estimated_duration=todo_dict.get("estimated_duration")
        )
    
    def todo_write(self, request: TodoWriteRequest) -> TodoWriteResult:
        """Create or update todos (matches Claude Code TodoWrite functionality)."""
        todos_created = 0
        todos_updated = 0
        
        try:
            for todo_dict in request.todos:
                todo_item = self._parse_todo_dict(todo_dict)
                
                # Check if todo already exists
                existing_todo = None
                for existing in self._session_todos:
                    if existing.id == todo_item.id:
                        existing_todo = existing
                        break
                
                if existing_todo:
                    # Update existing todo
                    existing_todo.content = todo_item.content
                    existing_todo.status = todo_item.status
                    existing_todo.priority = todo_item.priority
                    existing_todo.updated_at = datetime.now()
                    existing_todo.assigned_to = todo_item.assigned_to
                    existing_todo.tags = todo_item.tags
                    existing_todo.estimated_duration = todo_item.estimated_duration
                    todos_updated += 1
                else:
                    # Create new todo
                    self._session_todos.append(todo_item)
                    todos_created += 1
                
                # Also store in persistent dict for queries
                self._tasks[todo_item.id] = todo_item
            
            return TodoWriteResult(
                success=True,
                message=f"Successfully processed {len(request.todos)} todo items",
                todos_created=todos_created,
                todos_updated=todos_updated,
                total_todos=len(self._session_todos)
            )
            
        except Exception as e:
            return TodoWriteResult(
                success=False,
                message=f"Failed to process todos: {str(e)}",
                todos_created=0,
                todos_updated=0,
                total_todos=len(self._session_todos)
            )
    
    def delegate_task(self, request: TaskDelegationRequest) -> TaskDelegationResult:
        """Delegate a task to the most appropriate agent."""
        # Create a new task for the delegation
        task_id = self._generate_task_id()
        
        # Simple agent selection logic (can be enhanced with ManagerAgent integration)
        agent_scores = {
            "general-engineer": 0.5,
            "frontend-developer": 0.3,
            "backend-developer": 0.3,
            "data-scientist": 0.2,
            "devops-engineer": 0.4
        }
        
        # Analyze task description for keywords to improve selection
        task_lower = request.task_description.lower()
        
        # Frontend keywords
        if any(keyword in task_lower for keyword in ["react", "vue", "ui", "frontend", "css", "html"]):
            agent_scores["frontend-developer"] = 0.9
        
        # Backend keywords  
        elif any(keyword in task_lower for keyword in ["api", "backend", "server", "database", "sql"]):
            agent_scores["backend-developer"] = 0.9
        
        # Data keywords
        elif any(keyword in task_lower for keyword in ["data", "analysis", "ml", "model", "analytics"]):
            agent_scores["data-scientist"] = 0.9
            
        # DevOps keywords
        elif any(keyword in task_lower for keyword in ["deploy", "docker", "kubernetes", "ci/cd", "infrastructure"]):
            agent_scores["devops-engineer"] = 0.9
        
        # Use preferred agent if specified and boost its score
        if request.preferred_agent and request.preferred_agent in agent_scores:
            agent_scores[request.preferred_agent] = min(1.0, agent_scores[request.preferred_agent] + 0.3)
        
        # Select the agent with highest score
        selected_agent = max(agent_scores.items(), key=lambda x: x[1])
        
        # Create todo item for the delegated task
        delegated_todo = TodoItem(
            id=task_id,
            content=f"[DELEGATED] {request.task_description}",
            status=TaskStatus.PENDING,
            priority=request.priority,
            assigned_to=selected_agent[0],
            tags=["delegated"] + (["urgent"] if request.priority == TaskPriority.URGENT else []),
            estimated_duration=request.estimated_duration if hasattr(request, 'estimated_duration') else None
        )
        
        # Store the delegated task
        self._tasks[task_id] = delegated_todo
        self._session_todos.append(delegated_todo)
        
        # Generate reasoning
        reasoning = f"Selected {selected_agent[0]} based on task analysis. "
        if request.preferred_agent:
            reasoning += f"Preferred agent '{request.preferred_agent}' was considered. "
        reasoning += f"Confidence score: {selected_agent[1]:.2f}"
        
        # Estimate completion time (simple heuristic)
        estimated_completion = None
        if request.deadline:
            estimated_completion = request.deadline
        else:
            # Simple estimation based on priority
            hours_to_complete = {
                TaskPriority.URGENT: 2,
                TaskPriority.HIGH: 4, 
                TaskPriority.MEDIUM: 8,
                TaskPriority.LOW: 24
            }
            estimated_completion = datetime.now() + timedelta(hours=hours_to_complete[request.priority])
        
        return TaskDelegationResult(
            task_id=task_id,
            assigned_agent=selected_agent[0],
            confidence_score=selected_agent[1],
            reasoning=reasoning,
            estimated_completion=estimated_completion
        )
    
    def query_tasks(self, query: TaskQuery) -> TaskQueryResult:
        """Query tasks with filtering options."""
        matching_tasks = []
        
        for task in self._tasks.values():
            # Apply filters
            if query.status and task.status != query.status:
                continue
            if query.priority and task.priority != query.priority:
                continue
            if query.assigned_to and task.assigned_to != query.assigned_to:
                continue
            if query.tags and not any(tag in task.tags for tag in query.tags):
                continue
            
            matching_tasks.append(task)
        
        # Sort by priority and creation time
        priority_order = {TaskPriority.URGENT: 4, TaskPriority.HIGH: 3, TaskPriority.MEDIUM: 2, TaskPriority.LOW: 1}
        matching_tasks.sort(key=lambda t: (priority_order[t.priority], t.created_at), reverse=True)
        
        # Apply limit
        filtered_tasks = matching_tasks[:query.limit]
        
        return TaskQueryResult(
            tasks=filtered_tasks,
            total_count=len(matching_tasks),
            filtered_count=len(filtered_tasks)
        )


def create_task_management_toolset(config: Optional[CyroConfig] = None) -> FunctionToolset:
    """Create a task management toolset.
    
    Args:
        config: Cyro configuration
    
    Returns:
        FunctionToolset with task management tools
    """
    task_manager = TaskManager(config)
    
    # Create FunctionToolset for task management
    toolset = FunctionToolset()
    
    @toolset.tool
    def todo_write(request: TodoWriteRequest) -> TodoWriteResult:
        """Create or update a task list for tracking progress (matches Claude Code TodoWrite)."""
        return task_manager.todo_write(request)
    
    @toolset.tool
    def delegate_task(request: TaskDelegationRequest) -> TaskDelegationResult:
        """Delegate a task to the most appropriate agent based on requirements and expertise."""
        return task_manager.delegate_task(request)
    
    @toolset.tool
    def query_tasks(query: TaskQuery) -> TaskQueryResult:
        """Query and filter tasks by status, priority, assignment, or tags."""
        return task_manager.query_tasks(query)
    
    return toolset


def get_basic_task_tools(config: Optional[CyroConfig] = None) -> FunctionToolset:
    """Get basic task management tools.
    
    Args:
        config: Cyro configuration
        
    Returns:
        FunctionToolset with basic task management tools
    """
    return create_task_management_toolset(config)