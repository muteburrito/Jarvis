from langchain_huggingface import HuggingFaceEmbeddings
from langchain.text_splitter import RecursiveCharacterTextSplitter
from langchain_community.vectorstores import FAISS
from langchain.chains import RetrievalQA
from langchain_core.prompts import PromptTemplate
from langchain_community.document_loaders import PyPDFLoader # This is needed in utils.py

# Global to track conversation context
chat_history = []

def initialize_chain(documents=None, embed_model=None, llm=None):
    vectorstore = create_embeddings(documents, embed_model) if documents else None
    return create_qa_chain(vectorstore, llm)

def create_qa_chain(vectorstore=None, llm=None):
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

def split_docs(documents, chunk_size=1000, chunk_overlap=20):
    text_splitter = RecursiveCharacterTextSplitter(
        chunk_size=chunk_size,
        chunk_overlap=chunk_overlap
    )
    return text_splitter.split_documents(documents=documents)

def load_embedding_model(model_path, normalize_embedding=True):
    return HuggingFaceEmbeddings(
        model_name=model_path,
        model_kwargs={'device':'cpu'},
        encode_kwargs={'normalize_embeddings': normalize_embedding}
    )

def create_embeddings(chunks, embedding_model, storing_path="vectorstore"):
    vectorstore = FAISS.from_documents(chunks, embedding_model)
    vectorstore.save_local(storing_path)
    return vectorstore

def load_qa_chain(retriever, llm, prompt):
    return RetrievalQA.from_chain_type(
        llm=llm,
        retriever=retriever,
        chain_type="stuff",
        return_source_documents=True,
        chain_type_kwargs={'prompt': prompt}
    )

def get_response(query, chain, history_limit=10):
    global chat_history
    if 'forget everything' in query.lower() or 'forget that' in query.lower():
        chat_history = []  # Reset history
        formatted_prompt = {"query": "User: Hello!"}
    else:
        context = "\n".join(chat_history[-history_limit:])  # Keep only the last N interactions
        formatted_prompt = {"query": f"{context}\nUser: {query}"}
    
    try:
        response = chain(formatted_prompt)
        result = response if isinstance(response, str) else response.get('result', 'No result found')
        chat_history.append(f"User: {query}\nAI: {result}")
        return {'result': result}
    except Exception as e:
        return {'result': f'Error processing the query: {e}'}

# Template string for system responses
template = """
### System:
You are a respectful and honest assistant. Answer the user's \
questions using only the provided context. If you don't know the answer, \
just say you don't know. Provide references where possible.

### Context:
{context}

### User:
{question}

### Response:
"""
