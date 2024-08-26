import os
import shutil
import subprocess
import urllib
from werkzeug.utils import secure_filename
from lang_funcs import *
from langchain_community.llms import Ollama
from langchain_core.prompts import PromptTemplate

# Install and initialize dependencies
def check_and_install_ollama():
    try:
        subprocess.run(["ollama", "list"], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        print("Ollama is already installed.")
    except (subprocess.CalledProcessError, FileNotFoundError):
        print("Ollama not found, installing...")
        install_ollama()

def install_ollama():
    try:
        subprocess.run(["powershell", "-Command", 
                        "Invoke-WebRequest -Uri https://ollama.com/download/OllamaSetup.exe -OutFile OllamaSetup.exe"],
                        check=True)
        subprocess.run(["OllamaSetup.exe", "/quiet"], check=True)
        print("Ollama installed successfully.")
        os.remove("OllamaSetup.exe")
    except Exception as e:
        print(f"Failed to install Ollama: {e}")
        exit(1)

def check_and_copy_dll():
    dll_name = 'libomp140.x86_64.dll'
    system32_path = os.path.join(os.environ['WINDIR'], 'System32')
    target_dll_path = os.path.join(system32_path, dll_name)
    github_dll_url = 'https://github.com/muteburrito/pdf-chatbot/raw/main/Redist/libomp140.x86_64_x86-64/libomp140.x86_64.dll'

    if not os.path.exists(target_dll_path):
        try:
            with urllib.request.urlopen(github_dll_url) as response, open(dll_name, 'wb') as out_file:
                shutil.copyfileobj(response, out_file)
            shutil.move(dll_name, target_dll_path)
        except Exception as e:
            print(f"An error occurred: {e}")

def allowed_file(filename, allowed_extensions):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in allowed_extensions

def load_pdf_data(file_path):
    try:
        loader = PyMuPDFLoader(file_path=file_path)
        docs = loader.load()
        return docs
    except Exception as e:
        print(f"Error loading PDF {file_path}: {e}")
        return []

def load_and_process_documents(file_paths, upload_folder, allowed_extensions):
    documents = []
    for file_path in file_paths:
        if allowed_file(file_path, allowed_extensions):
            full_file_path = os.path.join(upload_folder, file_path)
            if file_path.endswith('.pdf'):
                docs = load_pdf_data(full_file_path)
            documents.extend(docs)
    return split_docs(documents)

def create_qa_chain(vectorstore=None, llm=None):
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
        return load_qa_chain(retriever, llm, prompt)
    else:
        def general_chat(inputs):
            query = inputs['query']
            prompt_text = prompt.format(context="", question=query)
            return llm.invoke(prompt_text)
        return general_chat

def initialize_chain(documents=None, embed_model=None, llm=None):
    vectorstore = create_embeddings(documents, embed_model) if documents else None
    return create_qa_chain(vectorstore, llm)
