@echo off
if not exist "chatbot" (
    echo "Creating venv..."
    python -m venv chatbot
    echo "Activating venv..."
    call chatbot\Scripts\activate
    echo "Installing deps..."
    pip install -r requirements.txt
)

if exist "__pycache__" (
    echo "Cleaning pycache before fresh run..."
    rmdir /s /q __pycache__
)

if exist "vectorstore" (
    echo "Cleaning old vectorstore"
    rmdir /s /q vectorstore
)

if exist "data" (
    echo "Cleaning old data dir"
    rmdir /s /q data
)

echo "Activating venv..."
call chatbot\Scripts\activate
echo "Starting App..."
python app.py
