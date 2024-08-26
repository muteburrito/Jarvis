$(document).ready(function() {

    // Set marked.js options
    marked.setOptions({
        breaks: true,  // Convert line breaks to <br> tags
        highlight: function (code, lang) {
            return hljs.highlightAuto(code).value;  // Use highlight.js for code highlighting
        }
    });

    // Load chat history if it exists in localStorage
    if (localStorage.getItem('chatHistory')) {
        $('#messages').html(localStorage.getItem('chatHistory'));
        $('.chat-box').scrollTop($('.chat-box')[0].scrollHeight);  // Scroll to the bottom
    }

    // Function to clear chat history
    $('#clearChatHistoryButton').click(function() {
        localStorage.removeItem('chatHistory');
        $('#messages').empty(); // Clear the messages from the UI as well
    });

    // Function to render markdown safely
    function renderMarkdown(markdownText) {
        return marked.parse(markdownText);
    }

    // We will use this to save chat history
    function updateChatHistory() {
        var chatHistory = $('#messages').html();
        localStorage.setItem('chatHistory', chatHistory);
    }

    $('#sendButton').click(function() {
        sendMessage();
    });

    $('#userInput').keypress(function(e) {
        if (e.which == 13) {
            sendMessage();
        }
    });

    // Automatically upload the files once selected
    $('#fileInput').change(function() {
        var files = $('#fileInput')[0].files; // Get the selected files
        if (files.length > 0) {
            for (var i = 0; i < files.length; i++) {
                var file = files[i];
                if (file.type === "application/pdf") {
                    uploadFile(file); // Upload each selected file
                } else {
                    alert("Only PDF files are allowed."); // Handle non-PDF files
                }
            }
        }
    });

    startTyping(); // Start the typing animation immediately on page load

    // Set interval for backspace and re-typing every 60 seconds
    setInterval(function() {
        backspaceAndRetype();
    }, 60000);

    function startTyping() {
        $('.title').css('width', '0');
        setTimeout(function() {
            $('.title').addClass('typing'); // Add typing class to animate
            setTimeout(function() {
                // Hide the cursor after typing finishes
                $('.title').css('border-right', 'none');
            }, 2500); // Adjust this timeout to match the typing duration
        }, 200); // Small delay to start animation
    }

    // Function for backspace and ReType
    function backspaceAndRetype() {
        $('.title').css('border-right', '4px solid #ff79c6'); // Show cursor again
        $('.title').css('width', '15ch'); // Ensure it's fully visible
        $('.title').removeClass('typing').css('animation', 'backspace 2s steps(15, end) forwards'); // Backspace animation
        
        setTimeout(function() {
            $('.title').css('animation', 'typing 1s steps(15, end) forwards').addClass('typing');
            setTimeout(function() {
                // Hide the cursor after retyping finishes
                $('.title').css('border-right', 'none');
            }, 2500); // Adjust this timeout to match the typing duration
        }, 2000); // Delay to allow backspacing to finish
    }

    // Function to show notifications
    function showNotification(message, type) {
        var notificationBar = $('.notification-bar');
        notificationBar.text(message); // Set the message text

        // Add the appropriate type class (success, error)
        if (type === 'success') {
            notificationBar.removeClass('error').addClass('success');
        } else if (type === 'error') {
            notificationBar.removeClass('success').addClass('error');
        }

        // Show the notification by adding the "show" class
        notificationBar.addClass('show');

        // Hide the notification after 3 seconds
        setTimeout(function() {
            notificationBar.removeClass('show');
        }, 3000); // You can adjust the time as needed
    }

    // Function to handle file upload
    function uploadFile(file) {
        var formData = new FormData();
        formData.append('file', file);

        $.ajax({
            url: '/upload',
            type: 'POST',
            data: formData,
            processData: false,
            contentType: false,
            success: function(response) {
                if (response.uploaded_files) {
                    var fileList = $('#fileList');
                    fileList.empty();  // Clear existing list
                    response.uploaded_files.forEach(function(file) {
                        fileList.append('<li>' + file + '</li>');  // Add new files
                    });
                }
            },
            error: function(error) {
                console.error('Error uploading files:', error);
            }
        });
    }

    // Fetch the list of uploaded files on page load
    fetchUploadedFiles();

    function fetchUploadedFiles() {
        $.ajax({
            url: '/list-files',
            type: 'GET',
            success: function(response) {
                if (response.uploaded_files) {
                    var fileList = $('#fileList');
                    fileList.empty();  // Clear existing list
                    response.uploaded_files.forEach(function(file) {
                        fileList.append('<li>' + file + '</li>');  // Add new files
                    });
                }
            },
            error: function(error) {
                console.error('Error fetching file list:', error);
            }
        });
    }

    // Process data button logic
    $('#processDataButton').click(function() {
        // Show the loading spinner
        $('#loading').show();

        $.ajax({
            url: '/process-data', // Backend endpoint to process the data and create vector store
            type: 'POST',
            success: function(response) {
                // Hide the loading spinner
                $('#loading').hide();
                
                showNotification('Data processed successfully, vector store created!', 'success');
            },
            error: function() {
                // Hide the loading spinner even if there's an error
                $('#loading').hide();

                showNotification('Error processing data.', 'error');
            }
        });
    });

    // Clear vector store button logic
    $('#clearVectorStoreButton').click(function() {
        $.ajax({
            url: '/clear-vector-store', // Backend endpoint to clear the vector store
            type: 'POST',
            success: function(response) {
                showNotification('Vector store cleared successfully!', 'success');
            },
            error: function() {
                showNotification('Error clearing vector store.', 'error');
            }
        });
    });

    // Clear files button logic
    $('#clearFilesButton').click(function() {
        $.ajax({
            url: '/clear-docs',
            type: 'POST',
            success: function(response) {
                $('#fileList').empty(); // Clear the file list in the sidebar
                showNotification('Documents cleared successfully!', 'success');
            },
            error: function() {
                showNotification('Error clearing documents.', 'error');
            }
        });
    });

    $('#sendButton').click(function() {
        sendMessage();
    });

    function sendMessage() {
        var userInput = $('#userInput').val();
        if (userInput) {
            $('#messages').append('<div class="message user-message">' + userInput + '</div>');
            $('#userInput').val('');
            
            updateChatHistory(); //Save User's message

            // Disable the button but do not change its inner HTML (so the icon stays intact)
            $('#sendButton').prop('disabled', true);
            $('#sendButton i').removeClass('fa-paper-plane').addClass('fa-spinner fa-spin'); // Change the icon to a spinner
    
            // Add three-dot animation
            $('#messages').append('<div class="message bot-message thinking-dots"><span class="thinking-dot">.</span><span class="thinking-dot">.</span><span class="thinking-dot">.</span></div>');
    
            $.ajax({
                url: '/ask',
                type: 'POST',
                contentType: 'application/json',
                data: JSON.stringify({ query: userInput }),
                success: function(response) {
                    // Remove the three-dot animation when response is received
                    $('.thinking-dots').remove();
                    
                    console.log(response.response);

                    // Safely parse the response using marked
                    var markdownResponse = renderMarkdown(response.response); // Parse Markdown response
                    
                    $('#messages').append('<div class="message bot-message">' + markdownResponse + '</div>');
                    
                    // Save chat history after bot response is added
                    updateChatHistory();

                    // Re-enable the button and restore the paper-plane icon
                    $('#sendButton').prop('disabled', false);
                    $('#sendButton i').removeClass('fa-spinner fa-spin').addClass('fa-paper-plane');
    
                    $('.chat-box').scrollTop($('.chat-box')[0].scrollHeight);
                },
                error: function() {
                    // Handle error and remove the animation
                    $('.thinking-dots').remove();
                    $('#messages').append('<div class="message bot-message">Error getting response.</div>');
                    
                    // Re-enable the button and restore the paper-plane icon
                    $('#sendButton').prop('disabled', false);
                    $('#sendButton i').removeClass('fa-spinner fa-spin').addClass('fa-paper-plane');
    
                    $('.chat-box').scrollTop($('.chat-box')[0].scrollHeight);
                }
            });
        }
    }
});
