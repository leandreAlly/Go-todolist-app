declare var Promise: any;

interface ArrayConstructor {
  from<T, U>(
    arrayLike: ArrayLike<T>,
    mapfn: (v: T, k: number) => U,
    thisArg?: any
  ): Array<U>;
  from<T>(arrayLike: ArrayLike<T>): Array<T>;
}

interface Todo {
  id: string;
  title: string;
  completed: boolean;
  createdAt: number;
}

interface ResponseData {
  message: string;
  data: Todo[];
}

interface CreateTodoResponse {
  message: string;
  dataID: string;
}

const localhostAddress = 'http://localhost:9000/todo';

const newTodoInput = document.querySelector(
  '#new-todo input'
) as HTMLInputElement;
let submitButton = document.querySelector('#submit') as HTMLButtonElement;

async function getTodos() {
  try {
    const response = await fetch(localhostAddress);
    const responseData: ResponseData = await response.json();
    console.log('result', responseData);

    return responseData.data;
  } catch (error) {
    console.log('Error:', error);
  }
}

async function displayTodos() {
  const todoList = await getTodos();

  if (typeof todoList === 'string') {
    console.error(todoList);
    return;
  }

  let todoListContainer = document.querySelector('#todos') as HTMLDivElement;
  todoListContainer.innerHTML = '';

  if (todoList?.length == 0) {
    todoListContainer.innerHTML += `
    <div class="todo">
        <span> You do not have any tasks </span>
    </div>
    `;
  } else {
    todoList?.forEach((todo) => {
      todoListContainer.innerHTML += `
      <div class="todo">
        <span>${todo.title} </span>

        <div class="actions">
           <button class="edit">
              <i class="fas fa-edit"></i>
          </button>
          <button data-id=${todo.id} class="delete">
             <i class="fas fa-trash-alt"></i>
          </button>
        </div>

        </div>
      `;
    });
  }

  deleteTaskButton();
}

async function createTodos(data: { title: string }) {
  try {
    const response = await fetch(localhostAddress, {
      method: 'POST',
      headers: {
        'Content-Type': 'Application/json',
      },
      body: JSON.stringify(data),
    });

    const result: CreateTodoResponse = await response.json();
    console.log('Success: ', result.message);
  } catch (error) {
    console.error('Error: ', error);
  }
}

async function deleteTodo(TodoID: string) {
  try {
    const response = await fetch(`${localhostAddress}/${TodoID}`, {
      method: 'DELETE',
    });
    const result = await response.json();
    console.log('Success: ', result.message);
  } catch (error) {
    console.error('Error: ', error);
  }
}

async function addTask() {
  const data = { title: newTodoInput.value };
  await createTodos(data);
  displayTodos();

  newTodoInput.value = '';
}

function deleteTaskButton() {
  const deleteTodoButtons: HTMLButtonElement[] = Array.from(
    document.querySelectorAll('.delete')
  );

  for (const deleteButton of deleteTodoButtons) {
    deleteButton.onclick = async function () {
      const todoID = deleteButton.getAttribute('data-id') || '';
      await deleteTodo(todoID);
      displayTodos();
    };
  }
}

displayTodos();

submitButton.addEventListener('click', () => addTask());
