import speech_recognition as sr
import pyttsx3
import requests
import keyboard
import time

# Initialize the recognizer
recognizer = sr.Recognizer()

# Initialize the text-to-speech engine
tts_engine = pyttsx3.init()

def listen_and_recognize():
    """
    This function listens to the user's voice, recognizes it, and converts it to text.
    """
    with sr.Microphone() as source:
        print("Listening... Press Ctrl + L to start speaking.")
        audio = recognizer.listen(source)
    
    try:
        # Convert speech to text
        query = recognizer.recognize_google(audio)
        print(f"You said: {query}")
        return query
    except sr.UnknownValueError:
        print("Sorry, I could not understand the audio.")
        return None
    except sr.RequestError as e:
        print(f"Could not request results; {e}")
        return None

def get_response_from_flask(query):
    """
    Sends the query to the Flask app and retrieves the response.
    """
    try:
        url = "http://localhost:5000/ask-voice"  # URL of the Flask app
        response = requests.post(url, json={"query": query})

        if response.status_code == 200:
            print("Response successfully processed by Flask app.")
            return True
        else:
            print("Error: Flask app did not return a successful response.")
            return False
    except Exception as e:
        print(f"Error while fetching response from Flask: {e}")
        return False

def main():
    print("Voice Assistant is running. Press Ctrl + L to ask a question.")

    while True:
        # Detect when Ctrl + L is pressed
        if keyboard.is_pressed('ctrl+l'):
            print("Ctrl + L detected. Recording your query...")
            
            # Listen and recognize the user's query
            query = listen_and_recognize()
            
            if query:
                # Send the query to the Flask app
                success = get_response_from_flask(query)
                
                if success:
                    print("Response was spoken by the Flask app.")
                else:
                    print("Failed to get a valid response from Flask.")
            
            # Pause to avoid multiple detections
            time.sleep(3)

if __name__ == "__main__":
    main()
