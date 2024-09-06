import speech_recognition as sr
import pyttsx3
import requests
import keyboard
import time
import os

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

def speak_response(response_text):
    """
    This function takes the text response and reads it aloud.
    """
    print(f"Jarvis says: {response_text}")
    tts_engine.say(response_text)
    tts_engine.runAndWait()

def get_response_from_flask(query):
    """
    Sends the query to the Flask app and retrieves the audio response.
    """
    try:
        url = "http://localhost:5000/ask-voice"  # URL of the Flask app
        response = requests.post(url, json={"query": query})

        if response.status_code == 200:
            audio_fp = f"{os.getcwd()}/response.mp3"
            with open(audio_fp, "wb") as f:
                f.write(response.content)
            return audio_fp
        else:
            print("Error: Flask app did not return a successful response.")
            return None
    except Exception as e:
        print(f"Error while fetching response from Flask: {e}")
        return None

def play_audio_response(audio_fp):
    """
    Plays the audio file returned by the Flask app.
    """
    if os.path.exists(audio_fp):
        os.system(f"start {audio_fp}")  # This will play the audio on Windows
    else:
        print("Audio file not found.")

def main():
    print("Voice Assistant is running. Press Ctrl + L to ask a question.")

    while True:
        # Detect when Ctrl + L is pressed
        if keyboard.is_pressed('ctrl+l'):
            print("Ctrl + L detected. Recording your query...")
            
            # Listen and recognize the user's query
            query = listen_and_recognize()
            
            if query:
                # Send the query to the Flask app and get the response
                audio_fp = get_response_from_flask(query)
                
                if audio_fp:
                    # Play the audio response from the Flask app
                    play_audio_response(audio_fp)
                else:
                    print("Failed to get a valid response from Flask.")
            
            # Pause to avoid multiple detections
            time.sleep(3)

if __name__ == "__main__":
    main()
