from langchain_huggingface import HuggingFaceEmbeddings
from langchain_community.document_loaders import PyMuPDFLoader
from langchain.text_splitter import RecursiveCharacterTextSplitter
from langchain_community.vectorstores import FAISS
from langchain.chains import RetrievalQA

def load_pdf_data(file_path):
    """
    Load a PDF document from the specified file path.

    Args:
    - file_path (str): Path to the PDF document.

    Returns:
    - Document: Loaded document.
    """
    # Creating a PyMuPDFLoader object with file_path
    loader = PyMuPDFLoader(file_path=file_path)
    
    # loading the PDF file
    docs = loader.load()
    
    # returning the loaded document
    return docs

def split_docs(documents, chunk_size=1000, chunk_overlap=20):
    """
    Split the documents into smaller chunks.

    Args:
    - documents (list): List of documents to split.
    - chunk_size (int, optional): Size of each chunk (default: 1000).
    - chunk_overlap (int, optional): Overlap between chunks (default: 20).

    Returns:
    - list: List of document chunks.
    """
    # Initializing the RecursiveCharacterTextSplitter with
    # chunk_size and chunk_overlap
    text_splitter = RecursiveCharacterTextSplitter(
        chunk_size=chunk_size,
        chunk_overlap=chunk_overlap
    )
    
    # Splitting the documents into chunks
    chunks = text_splitter.split_documents(documents=documents)
    
    # returning the document chunks
    return chunks

def load_embedding_model(model_path, normalize_embedding=True):
    """
    Load a text embedding model.

    Args:
    - model_path (str): Path to the embedding model.
    - normalize_embedding (bool, optional): Whether to normalize embeddings (default: True).

    Returns:
    - HuggingFaceEmbeddings: Text embedding model.
    """
    return HuggingFaceEmbeddings(
        model_name=model_path,
        model_kwargs={'device':'cpu'}, # here we will run the model with CPU only
        encode_kwargs = {
            'normalize_embeddings': normalize_embedding # keep True to compute cosine similarity
        }
    )

def create_embeddings(chunks, embedding_model, storing_path="vectorstore"):
    """
    Create embeddings for document chunks.

    Args:
    - chunks (list): List of document chunks.
    - embedding_model (HuggingFaceEmbeddings): Text embedding model.
    - storing_path (str, optional): Path to store the embeddings (default: "vectorstore").

    Returns:
    - FAISS: Vector store containing embeddings.
    """
    # Creating the embeddings using FAISS
    vectorstore = FAISS.from_documents(chunks, embedding_model)
    
    # Saving the model in current directory
    vectorstore.save_local(storing_path)
    
    # returning the vectorstore
    return vectorstore

def load_qa_chain(retriever, llm, prompt):
    """
    Create a question-answering chain.

    Args:
    - retriever (FAISS): Vector store retriever.
    - llm: Language model.
    - prompt (str): Prompt template.

    Returns:
    - RetrievalQA: Question-answering chain.
    """
    return RetrievalQA.from_chain_type(
        llm=llm,
        retriever=retriever, # here we are using the vectorstore as a retriever
        chain_type="stuff",
        return_source_documents=True, # including source documents in output
        chain_type_kwargs={'prompt': prompt} # customizing the prompt
    )

def get_response(query, chain):
    """
    Retrieve response for a given query using the question-answering chain.

    Args:
    - query (str): Question query.
    - chain (RetrievalQA): Question-answering chain.

    Returns:
    - str: Response to the query.
    """
    # Getting response from chain
    response = chain.invoke({'query': query})
    
    # Retrieve the result and remove any newlines
    result_text = response['result']
    
    # Remove newlines and format the text for better readability
    formatted_text = result_text.replace('\n', ' ').strip()
    
    return formatted_text

# Prompt template
prompt_template = """
### System:
You are an AI Assistant that follows instructions extremely well. \
Help as much as you can.

### User:
{prompt}

### Response:

"""

# Template string for system responses
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