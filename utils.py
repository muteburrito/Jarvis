import os
import shutil
import subprocess
import urllib
from lang_funcs import *

# Install and initialize dependencies
def check_and_install_ollama():
    if not is_ollama_installed():
        install_ollama()

def is_ollama_installed():
    try:
        subprocess.run(["ollama", "list"], check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        return True
    except (subprocess.CalledProcessError, FileNotFoundError):
        return False

def install_ollama():
    try:
        subprocess.run(["powershell", "-Command", 
                        "Invoke-WebRequest -Uri https://ollama.com/download/OllamaSetup.exe -OutFile OllamaSetup.exe"],
                        check=True)
        subprocess.run(["OllamaSetup.exe", "/quiet"], check=True)
        os.remove("OllamaSetup.exe")
    except Exception as e:
        print(f"Failed to install Ollama: {e}")
        exit(1)

def pull_llama_model(model_name="llama3.1"):
    try:
        subprocess.run(["ollama", "pull", model_name], check=True)
    except subprocess.CalledProcessError:
        print(f"Failed to pull model {model_name}.")
        exit(1)

def check_and_copy_dll(dll_name='libomp140.x86_64.dll'):
    system32_path = os.path.join(os.environ['WINDIR'], 'System32')
    target_dll_path = os.path.join(system32_path, dll_name)
    github_dll_url = f'https://github.com/muteburrito/pdf-chatbot/raw/main/Redist/libomp140.x86_64_x86-64/{dll_name}'

    if not os.path.exists(target_dll_path):
        download_and_copy_dll(github_dll_url, target_dll_path)

def download_and_copy_dll(url, target_path):
    try:
        with urllib.request.urlopen(url) as response, open(os.path.basename(target_path), 'wb') as out_file:
            shutil.copyfileobj(response, out_file)
        shutil.move(os.path.basename(target_path), target_path)
    except Exception as e:
        print(f"An error occurred: {e}")

def load_pdf_data_if_allowed(filename, allowed_extensions, upload_folder):
    if allowed_file(filename, allowed_extensions):
        file_path = os.path.join(upload_folder, filename)
        return load_pdf_data(file_path)
    return []

def load_pdf_data(file_path):
    try:
        loader = PyPDFLoader(file_path=file_path)  # Reverting to PyPDFLoader
        docs = loader.load()
        return docs
    except Exception as e:
        print(f"Error loading PDF {file_path}: {e}")
        return []

def allowed_file(filename, allowed_extensions):
    return '.' in filename and filename.rsplit('.', 1)[1].lower() in allowed_extensions
