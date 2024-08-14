@echo off
if not exist "chatbot" (
    echo "Creating venv..."
    python -m venv chatbot
    echo "Activating venv..."
    call chatbot\Scripts\activate
    echo "Installing deps..."
    pip install -r requirements.txt
)
echo "Activating venv..."
call chatbot\Scripts\activate
echo "Starting App..."
python app.py
