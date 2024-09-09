import os
import pyttsx3
from gtts import gTTS 
import threading
import shutil
from flask import Flask, request, jsonify, render_template, send_file
from werkzeug.utils import secure_filename
from flask_cors import CORS
from utils import *
from langchain_community.llms import Ollama

# Flask app setup
app = Flask(__name__)
CORS(app)

# Initialize pyttsx3 TTS engine
tts_engine = pyttsx3.init()

UPLOAD_FOLDER = 'data'
ALLOWED_EXTENSIONS = {'pdf'}
app.config['UPLOAD_FOLDER'] = UPLOAD_FOLDER

# List to track uploaded files
uploaded_files = []

# Initialize the models and environment
check_and_install_ollama()
pull_llama_model()
check_and_copy_dll()

llm = Ollama(model="llama3.1", temperature=0)
embed = load_embedding_model(model_path="all-MiniLM-L6-v2")

# Initialize the chain
chain = initialize_chain(embed_model=embed, llm=llm)

# Routes

@app.route('/')
def index():
    data_directory = app.config['UPLOAD_FOLDER']
    os.makedirs(data_directory, exist_ok=True)
    files = [f for f in os.listdir(data_directory) if allowed_file(f, ALLOWED_EXTENSIONS)]
    return render_template('index.html', files=files)

@app.route('/ask', methods=['POST'])
def ask():
    data = request.json
    query = data.get('query')
    if not query:
        return jsonify({'error': 'No query provided'}), 400
    try:
        # Process the query with chat history retained
        response = get_response(query, chain)
        result = response if isinstance(response, str) else response.get('result', 'No result found')
        return jsonify({'response': str(result)})
    except Exception as e:
        return jsonify({'error': 'Error processing request'}), 500

@app.route('/ask-voice', methods=['POST'])
def ask_voice():
    data = request.json
    query = data.get('query')
    if not query:
        return jsonify({'error': 'No query provided'}), 400

    try:
        response = get_response(query, chain)  # Process the query with the language model
        result = response.get('result', 'No result found')

        # Function to run the TTS in a separate thread
        def speak_text(text):
            # Configure pyttsx3 for Jarvis-like voice settings (optional)
            voices = tts_engine.getProperty('voices')
            tts_engine.setProperty('voice', voices[1].id)  # Change to another voice index if needed
            tts_engine.setProperty('rate', 190)  # Set the rate of speech

            # Speak the result directly (without saving to file)
            tts_engine.say(text)
            tts_engine.runAndWait()

        # Start the TTS in a separate thread so that it doesn't block
        tts_thread = threading.Thread(target=speak_text, args=(result,))
        tts_thread.start()

        # Return the response immediately
        return jsonify({'message': 'Response spoken successfully!'}), 200

    except Exception as e:
        print(f"Error processing request: {e}")  # Log the error
        return jsonify({'error': 'Error processing request'}), 500

@app.route('/list-files', methods=['GET'])
def list_files():
    try:
        files = os.listdir(app.config['UPLOAD_FOLDER'])
        pdf_files = [file for file in files if file.endswith('.pdf')]
        return jsonify({'uploaded_files': pdf_files})
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/upload', methods=['POST'])
def upload_file():
    # Ensure the upload folder (data) exists
    os.makedirs(app.config['UPLOAD_FOLDER'], exist_ok=True)

    # Check if 'file' is part of the request
    if 'file' not in request.files:
        return jsonify({'error': 'No file part'}), 400

    files = request.files.getlist('file')  # Multiple files can be uploaded
    if not files or files[0].filename == '':
        return jsonify({'error': 'No selected file'}), 400

    for file in files:
        if file and allowed_file(file.filename, ALLOWED_EXTENSIONS):
            # Secure the filename
            filename = secure_filename(file.filename)

            # Define the file path
            file_path = os.path.join(app.config['UPLOAD_FOLDER'], filename)

            # Save the file to the specified folder
            file.save(file_path)

            # Append the filename to the uploaded_files list
            if filename not in uploaded_files:
                uploaded_files.append(filename)

    return jsonify({'message': 'Files uploaded and saved successfully!', 'uploaded_files': uploaded_files})

@app.route('/process-data', methods=['POST'])
def process_data():
    global chain
    data_directory = app.config['UPLOAD_FOLDER']
    
    if not os.path.exists(data_directory):
        return jsonify({'error': 'Data directory does not exist'}), 400
    
    # Get all PDF files in the data directory
    files = [f for f in os.listdir(data_directory) if allowed_file(f, ALLOWED_EXTENSIONS)]
    
    if not files:
        return jsonify({'error': 'No files to process'}), 400

    # Load and process documents directly from the file paths
    documents = load_and_process_documents(files, data_directory, ALLOWED_EXTENSIONS)

    if not documents:
        return jsonify({'error': 'No valid documents found to process'}), 400

    # Initialize the chain with the processed documents
    chain = initialize_chain(documents, embed_model=embed, llm=llm)
    
    return jsonify({'message': 'Vector store created successfully!'})

@app.route('/clear-vector-store', methods=['POST'])
def clear_vector_store():
    global chain
    vectorstore_path = 'vectorstore'
    if os.path.exists(vectorstore_path):
        shutil.rmtree(vectorstore_path)
    chain = initialize_chain(embed_model=embed, llm=llm)
    return jsonify({'message': 'Vector store cleared successfully!'})

@app.route('/clear-docs', methods=['POST'])
def clear_documents():
    global uploaded_files
    data_path = 'data'
    uploaded_files = []
    if os.path.exists(data_path):
        shutil.rmtree(data_path)
    return jsonify({'message': 'Data cleared and chat reset!'})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
