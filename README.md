---
runme:
  id: 01HSTE2B7Y9XRAM6NGF9N9EBWN
  version: v3
---

# Jarvis

## Introduction

Jarvis is a simple RAG (Retrieval Augmented Generation) based QnA bot which is using Llama2 for the communication, all-MiniLM-L6-v2 for creating embeddings.
The Bot also uses Ollama for loading the base model in this case Llama2 and chains from Langchains for the question and answer.

## Installation

Head over to the [Releases Page](http://13.201.88.74:8080/job/Jarvis-Pipeline/) and grab the latest successful exe. This exe contains all the required dependancies, and if you don't have them, the server exe will install it for you.

Note- Keep the EXE in a folder, so that the vectorstore and data folder are properly kept in one place.

## Code Setup

You need to have [Python 3.11](https://www.python.org/downloads/release/python-3118/) installled on your system, the best way would be to create a [virtual enviornemnt](https://docs.python.org/3/library/venv.html).
To simplify things we have a [setup_workspace](setup_workspace.cmd) batch file, which will help you setup the workspace.
We also have, a [run_webapp](run_webapp.cmd) batch file, which will run the Web App for you.

If you are in VS Code, I would recommend you to have the following extensions installed-

- [Python](https://marketplace.visualstudio.com/items?itemName=ms-python.python)
- [Pylance](https://marketplace.visualstudio.com/items?itemName=ms-python.vscode-pylance)
- [Jupyter](https://marketplace.visualstudio.com/items?itemName=ms-toolsai.jupyter)
- [Runme](https://marketplace.visualstudio.com/items?itemName=stateful.runme) (This is neeed to visualize the documents like the [README.md](README.md))

Note: If the app gives you error regarding `fbgemm.dll` or one of its dependencies please first install the [VCRedist](Redist\VC_redist.x64.exe), and if that also does not fix the probelm, simply copy paste the dll given in the Redist folder to C:\Windows\System32

## References

For more information on how the code works, I would highly recommend to check out the [LangChain](https://python.langchain.com/docs/get_started/introduction) documentation page.
For specific topics check below-

- [LangChain Doc used in this Bot](https://python.langchain.com/docs/get_started/quickstart)
- [Huggingface embeddings](https://python.langchain.com/docs/integrations/platforms/huggingface)
- [Huggingface embeddings-2](https://python.langchain.com/docs/integrations/text_embedding/huggingfacehub)
- [PyMuPDF](https://python.langchain.com/docs/modules/data_connection/document_loaders/pdf)
- [Recursive Character Text Splitter](https://python.langchain.com/docs/modules/data_connection/document_transformers/recursive_text_splitter)
- [FAISS (FaceBook AI Similarity Search)](https://python.langchain.com/docs/integrations/vectorstores/faiss)
- [RetrievalQA](https://api.python.langchain.com/en/latest/chains/langchain.chains.retrieval_qa.base.RetrievalQA.html#langchain.chains.retrieval_qa.base.RetrievalQA)
- For more info on Chains check out [this page](https://python.langchain.com/docs/modules/chains/#legacy-chains)
- [What is a Base Retriever ?](https://api.python.langchain.com/en/latest/retrievers/langchain_core.retrievers.BaseRetriever.html#langchain_core.retrievers.BaseRetriever)
- [all-MiniLM-L6-V2](https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2)
- [Llama2](https://huggingface.co/meta-llama/Llama-2-7b)
- [Ollama](https://ollama.com/)
- [What is RAG?](https://www.databricks.com/glossary/retrieval-augmented-generation-rag)
