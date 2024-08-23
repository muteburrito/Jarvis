$(document).ready(function() {
    $('#sendButton').click(function() {
        sendMessage();
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

    function backspaceAndRetype() {
        $('.title').css('border-right', '4px solid #ff79c6'); // Show cursor again
        $('.title').css('width', '15ch'); // Ensure it's fully visible
        $('.title').removeClass('typing').css('animation', 'backspace 2s steps(15, end) forwards'); // Backspace animation
        
        setTimeout(function() {
            $('.title').css('animation', 'typing 2s steps(15, end) forwards').addClass('typing');
            setTimeout(function() {
                // Hide the cursor after retyping finishes
                $('.title').css('border-right', 'none');
            }, 2500); // Adjust this timeout to match the typing duration
        }, 2000); // Delay to allow backspacing to finish
    }

    $('#userInput').keypress(function(e) {
        if (e.which == 13) {
            sendMessage();
        }
    });

    // File input change event to show selected file name
    $('#fileInput').change(function() {
        var file = $('#fileInput')[0].files[0];
        if (file) {
            $('#fileNameDisplay').text(file.name); // Show selected file name
            $('#uploadButton').prop('disabled', false); // Enable Upload File button when a file is selected
        } else {
            $('#fileNameDisplay').text('No file chosen');
            $('#uploadButton').prop('disabled', true); // Disable Upload File button if no file is selected
        }
    });

    // Ensure file is properly selected before uploading
    $('#uploadButton').click(function() {
        var fileInput = $('#fileInput')[0];
        if (fileInput.files.length === 0) {
            showNotification('No file selected for upload.', 'error');
            return;
        }

        var formData = new FormData();
        formData.append('file', fileInput.files[0]);

        $.ajax({
            url: '/upload',
            type: 'POST',
            data: formData,
            processData: false,
            contentType: false,
            beforeSend: function() {
                $('#uploadButton').prop('disabled', true).text('Uploading...');
            },
            success: function(response) {
                showNotification('File uploaded and processed successfully!', 'success');
                $('#uploadButton').prop('disabled', false).text('Upload File');
                $('#fileInput').val(''); // Clear file input after successful upload
                $('#fileNameDisplay').text('No file chosen'); // Reset file name display
            },
            error: function() {
                showNotification('Error uploading file.', 'error');
                $('#uploadButton').prop('disabled', false).text('Upload File');
            }
        });
    });

    $('#clearButton').click(function() {
        $.ajax({
            url: '/clear',
            type: 'POST',
            success: function(response) {
                showNotification('Documents cleared and chat reset.', 'success');
                $('#messages').html(''); // Clear the chat messages
                $('#userInput').val('');
                $('.chat-box').scrollTop($('.chat-box')[0].scrollHeight);
            },
            error: function() {
                showNotification('Error clearing documents.', 'error');
            }
        });
    });

    function sendMessage() {
        var userInput = $('#userInput').val();
        if (userInput) {
            $('#messages').append('<div class="message user-message">' + userInput + '</div>');
            $('#userInput').val('');
            $('#sendButton').prop('disabled', true).text('Sending...');

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
                    
                    // Safely parse the response using marked
                    var markdownResponse = marked.parse(response.response); // Parse Markdown response
                    
                    $('#messages').append('<div class="message bot-message">' + markdownResponse + '</div>');
                    $('#sendButton').prop('disabled', false).text('Send');
                    $('.chat-box').scrollTop($('.chat-box')[0].scrollHeight);
                },
                error: function() {
                    // Handle error and remove the animation
                    $('.thinking-dots').remove();
                    $('#messages').append('<div class="message bot-message">Error getting response.</div>');
                    $('#sendButton').prop('disabled', false).text('Send');
                    $('.chat-box').scrollTop($('.chat-box')[0].scrollHeight);
                }
            });
        }
    }

    function showNotification(message, type) {
        var notification = $('<div class="notification-bar ' + type + '">' + message + '</div>');
        $('body').append(notification);
        notification.addClass('show');

        // Hide notification after 3 seconds
        setTimeout(function() {
            notification.removeClass('show');
            setTimeout(function() {
                notification.remove();
            }, 300); // Delay to allow for CSS transition
        }, 3000);
    }
});
