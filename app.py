import os
import pyttsx3
from gtts import gTTS 
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
        response = get_response(query, chain)
        result = response if isinstance(response, str) else response.get('result', 'No result found')
        return jsonify({'response': str(result)})
    except Exception as e:
        return jsonify({'error': f'Error processing request: {e}'}), 500
    
@app.route('/speak-response', methods=['POST'])
def speak_response():
    data = request.json
    response_text = data.get('response_text')
    if not response_text:
        return jsonify({'error': 'No response text provided'}), 400

    try:
        tts = gTTS(text=response_text, lang='en')
        audio_file_path = os.path.join(app.config['UPLOAD_FOLDER'], 'response.mp3')
        tts.save(audio_file_path)
        return send_file(audio_file_path, mimetype='audio/mpeg')
    except Exception as e:
        return jsonify({'error': f'Error processing request: {e}'}), 500

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
    os.makedirs(app.config['UPLOAD_FOLDER'], exist_ok=True)
    if 'file' not in request.files:
        return jsonify({'error': 'No file part'}), 400

    files = request.files.getlist('file')
    if not files or files[0].filename == '':
        return jsonify({'error': 'No selected file'}), 400

    for file in files:
        if file and allowed_file(file.filename, ALLOWED_EXTENSIONS):
            filename = secure_filename(file.filename)
            file_path = os.path.join(app.config['UPLOAD_FOLDER'], filename)
            file.save(file_path)
            if filename not in uploaded_files:
                uploaded_files.append(filename)

    return jsonify({'message': 'Files uploaded and saved successfully!', 'uploaded_files': uploaded_files})

@app.route('/process-data', methods=['POST'])
def process_data():
    global chain
    data_directory = app.config['UPLOAD_FOLDER']
    
    if not os.path.exists(data_directory):
        return jsonify({'error': 'Data directory does not exist'}), 400
    
    files = [f for f in os.listdir(data_directory) if allowed_file(f, ALLOWED_EXTENSIONS)]
    
    if not files:
        return jsonify({'error': 'No files to process'}), 400

    documents = load_and_process_documents(files, data_directory, ALLOWED_EXTENSIONS)

    if not documents:
        return jsonify({'error': 'No valid documents found to process'}), 400

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
