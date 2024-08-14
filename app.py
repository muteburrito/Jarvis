from flask import Flask, request, jsonify, render_template
from werkzeug.utils import secure_filename
from lang_funcs import *
from langchain_community.llms import Ollama
from langchain_core.prompts import PromptTemplate
import os
from docx import Document as DocxDocument
import pandas as pd
from flask_cors import CORS

app = Flask(__name__)
CORS(app)

# Load the LLM model
llm = Ollama(model="llama3", temperature=0)

# Load the Embedding Model
embed = load_embedding_model(model_path="all-MiniLM-L6-v2")

UPLOAD_FOLDER = 'data'
ALLOWED_EXTENSIONS = {'pdf', 'docx', 'xlsx'}
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
        return [{'content': content}]
    except Exception as e:
        print(f"Error loading Excel file {file_path}: {e}")
        return []

def load_and_process_documents(data_folder):
    documents = []
    for filename in os.listdir(data_folder):
        file_path = os.path.join(data_folder, filename)
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

# Initialize documents and vector store
documents = []
vectorstore = None

def initialize_vectorstore():
    global documents, vectorstore
    documents = load_and_process_documents(app.config['UPLOAD_FOLDER'])
    vectorstore = create_embeddings(documents, embed)
    return vectorstore

# Initialize vectorstore on startup
initialize_vectorstore()

# Create the QA chain
def create_qa_chain(vectorstore):
    template = """
    ### System:
    You are a respectful and honest assistant. You have to answer the user's \
    questions using only the context provided to you. If you don't know the answer, \
    just say you don't know. Don't try to make up an answer. Also try to give reference for your answer.

    ### Context:
    {context}

    ### User:
    {question}

    ### Response:
    """
    prompt = PromptTemplate.from_template(template)
    retriever = vectorstore.as_retriever()
    return load_qa_chain(retriever, llm, prompt)

# Initialize the chain with the initial vectorstore
chain = create_qa_chain(vectorstore)

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/ask', methods=['POST'])
def ask():
    data = request.json
    query = data.get('query')
    if not query:
        return jsonify({'error': 'No query provided'}), 400

    global chain
    response = get_response(query, chain)
    return jsonify({'response': response})

@app.route('/upload', methods=['POST'])
def upload_file():
    if 'file' not in request.files:
        return jsonify({'error': 'No file part'}), 400

    file = request.files['file']
    if file.filename == '':
        return jsonify({'error': 'No selected file'}), 400

    if file and allowed_file(file.filename):
        filename = secure_filename(file.filename)
        filepath = os.path.join(app.config['UPLOAD_FOLDER'], filename)
        file.save(filepath)
        
        # Reprocess the vector store with the new document
        global vectorstore, chain
        new_docs = load_and_process_documents(app.config['UPLOAD_FOLDER'])
        vectorstore = create_embeddings(new_docs, embed)
        chain = create_qa_chain(vectorstore)
        
        return jsonify({'message': 'File uploaded and processed successfully!'})
    else:
        return jsonify({'error': 'Invalid file format'}), 400

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5000)
