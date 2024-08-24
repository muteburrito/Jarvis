import os
import shutil
import subprocess
import urllib
from flask import Flask, request, jsonify, render_template, send_from_directory
from werkzeug.utils import secure_filename
from lang_funcs import *
from langchain_community.llms import Ollama
from langchain_core.prompts import PromptTemplate
from docx import Document as DocxDocument
import pandas as pd
from flask_cors import CORS

app = Flask(__name__)
CORS(app)

def check_and_copy_dll():
    dll_name = 'libomp140.x86_64.dll'
    system32_path = os.path.join(os.environ['WINDIR'], 'System32')
    target_dll_path = os.path.join(system32_path, dll_name)
    github_dll_url = 'https://github.com/muteburrito/pdf-chatbot/raw/main/Redist/libomp140.x86_64_x86-64/libomp140.x86_64.dll'

    # Check if DLL is present in System32
    if not os.path.exists(target_dll_path):
        print(f"{dll_name} not found in {system32_path}. Downloading from GitHub...")

        try:
            # Download the DLL from the GitHub URL
            with urllib.request.urlopen(github_dll_url) as response, open(dll_name, 'wb') as out_file:
                shutil.copyfileobj(response, out_file)
            # Move the DLL to System32
            shutil.move(dll_name, target_dll_path)
            print(f"{dll_name} successfully downloaded and copied to {system32_path}.")
        except PermissionError:
            print(f"Permission denied. Run the exe with administrator privileges to copy {dll_name} to System32.")
        except Exception as e:
            print(f"An error occurred: {e}")
    else:
        print(f"{dll_name} is already present in {system32_path}.")

def check_and_install_ollama():
    try:
        # Check if Ollama is installed by running "ollama list"
        subprocess.run(["ollama", "list"], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        print("Ollama is already installed.")
    except (subprocess.CalledProcessError, FileNotFoundError):
        print("Ollama not found, installing...")
        install_ollama()

def install_ollama():
    try:
        # Download and run Ollama installer
        subprocess.run(["powershell", "-Command", 
                        "Invoke-WebRequest -Uri https://ollama.com/download/OllamaSetup.exe -OutFile OllamaSetup.exe"],
                        check=True)
        subprocess.run(["OllamaSetup.exe", "/quiet"], check=True)
        print("Ollama installed successfully.")
        # Clean up installer file
        os.remove("OllamaSetup.exe")
    except Exception as e:
        print(f"Failed to install Ollama: {e}")
        exit(1)

def pull_llama_model():
    try:
        # Pull the Llama 3.1 model using Ollama
        subprocess.run(["ollama", "pull", "llama3.1"], check=True)
        print("Llama 3.1 model pulled successfully.")
    except subprocess.CalledProcessError:
        print("Failed to pull Llama 3.1 model.")
        exit(1)

# Before starting the app, check if Ollama is installed and pull the model
check_and_install_ollama()
pull_llama_model()
# Call the function before the app starts
check_and_copy_dll()

# Load the LLM model
llm = Ollama(model="llama3.1", temperature=0)

# Load the Embedding Model
embed = load_embedding_model(model_path="all-MiniLM-L6-v2")

UPLOAD_FOLDER = 'data'
ALLOWED_EXTENSIONS = {'pdf'} # Removing other file types untill we add a feature to split text and create chunks for xlsx, doc, and ppt
app.config['UPLOAD_FOLDER'] = UPLOAD_FOLDER

def allowed_file(filename):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in ALLOWED_EXTENSIONS

def load_pdf_data(file_path):
    try:
        loader = PyMuPDFLoader(file_path=file_path)
        docs = loader.load()
        return docs
    except Exception as e:
        print(f"Error loading PDF {file_path}: {e}")
        return []

def load_word_data(file_path):
    try:
        doc = DocxDocument(file_path)
        full_text = []
        for para in doc.paragraphs:
            full_text.append(para.text)
        return [{'content': '\n'.join(full_text)}]
    except Exception as e:
        print(f"Error loading Word document {file_path}: {e}")
        return []

def load_excel_data(file_path):
    try:
        excel_data = pd.read_excel(file_path)
        content = excel_data.to_string(index=False)
        return [{'page_content': content}]
    except Exception as e:
        print(f"Error loading Excel file {file_path}: {e}")
        return []

def load_and_process_documents(files):
    documents = []
    for file in files:
        if file and allowed_file(file.filename):
            filename = secure_filename(file.filename)
            file_path = os.path.join(app.config['UPLOAD_FOLDER'], filename)

            # Ensure the upload folder exists
            os.makedirs(app.config['UPLOAD_FOLDER'], exist_ok=True)

            file.save(file_path)

            if filename.endswith('.pdf'):
                docs = load_pdf_data(file_path)
            elif filename.endswith('.docx'):
                docs = load_word_data(file_path)
            elif filename.endswith('.xlsx'):
                docs = load_excel_data(file_path)
            else:
                print(f"Unsupported file type: {filename}")
                continue
            documents.extend(docs)
    
    return split_docs(documents)

def create_qa_chain(vectorstore=None):
    template = """
    ### System:
    You are a respectful and honest assistant. You answer the user's questions using the context provided when available. If no context is provided, answer the questions generally. If you don't know the answer, just say you don't know.

    ### Context:
    {context}

    ### User:
    {question}

    ### Response:
    """
    prompt = PromptTemplate.from_template(template)
    
    if vectorstore:
        retriever = vectorstore.as_retriever()
        return load_qa_chain(retriever, llm, prompt) # Need to use the new function to get rid of the deprecated warning. The function is still WIP
    else:
        def general_chat(inputs):
            query = inputs['query']
            prompt_text = prompt.format(context="", question=query)
            return llm.invoke(prompt_text)
        
        return general_chat

def initialize_chain(documents=None):
    vectorstore = create_embeddings(documents, embed) if documents else None
    return create_qa_chain(vectorstore)

# Initially, start with no documents
chain = initialize_chain()

@app.route('/favicon.ico')
def favicon():
    return send_from_directory(os.path.join(app.root_path, 'static'),
                               'favicon.ico', mimetype='image/vnd.microsoft.icon')

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/ask', methods=['POST'])
def ask():
    data = request.json
    query = data.get('query')
    if not query:
        return jsonify({'error': 'No query provided'}), 400

    try:
        # Get the response from the QA chain
        response = get_response(query, chain)
        
        # Debugging: Print the response to inspect it
        # print(f"Response: {response}")

        # Handle cases where response might be a string
        if isinstance(response, dict):
            result = response.get('result', 'No result found')
        elif isinstance(response, str):
            result = response
        else:
            result = 'Unexpected response format'

        # Ensure the result is JSON serializable
        if isinstance(result, dict):
            json_response = {k: str(v) for k, v in result.items()}
        elif isinstance(result, list):
            json_response = [str(item) for item in result]
        else:
            json_response = str(result)

        return jsonify({'response': json_response})

    except Exception as e:
        # Print the exception details for debugging
        print(f"Exception: {e}")
        return jsonify({'error': 'Error processing request'}), 500

@app.route('/upload', methods=['POST'])
def upload_file():
    if 'file' not in request.files:
        return jsonify({'error': 'No file part'}), 400

    files = request.files.getlist('file')
    if not files or files[0].filename == '':
        return jsonify({'error': 'No selected file'}), 400

    documents = load_and_process_documents(files)
    
    global chain
    chain = initialize_chain(documents)
    
    return jsonify({'message': 'Files uploaded and processed successfully!'})

@app.route('/clear', methods=['POST'])
def clear_documents():
    global chain
    
    # Define paths
    data_path = 'data'
    vectorstore_path = 'vectorstore'
    
    # Helper function to delete files/directories
    def delete_path(path):
        if os.path.exists(path):
            try:
                if os.path.isfile(path):
                    os.remove(path)
                elif os.path.isdir(path):
                    shutil.rmtree(path)
            except PermissionError as e:
                print(f"Permission error: {e}")
                return jsonify({'message': f'Failed to delete {path} due to permission issues.'}), 500
        else:
            print(f"Path not found: {path}")
    
    # Delete data and vectorstore
    delete_path(data_path)
    delete_path(vectorstore_path)
    
    # Reinitialize the chain with no documents
    chain = initialize_chain()
    
    return jsonify({'message': 'Data and vectorstore cleared and chat reset!'})

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
