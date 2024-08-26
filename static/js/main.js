$(document).ready(function() {

    // Set marked.js options
    marked.setOptions({
        breaks: true,  // Convert line breaks to <br> tags
        highlight: function (code, lang) {
            return hljs.highlightAuto(code).value;  // Use highlight.js for code highlighting
        }
    });

    // Function to render markdown safely
    function renderMarkdown(markdownText) {
        return marked.parse(markdownText);
    }

    // Clear files from sidebar
    $('#clearFilesButton').click(function() {
        // Clear uploaded files and reset sidebar
        $.ajax({
            url: '/clear',
            type: 'POST',
            success: function() {
                $('#fileList').empty(); // Clear sidebar list
                showNotification('Documents cleared successfully', 'success');
            },
            error: function() {
                showNotification('Error clearing documents', 'error');
            }
        });
    });

    $('#sendButton').click(function() {
        sendMessage();
    });

    $('#userInput').keypress(function(e) {
        if (e.which == 13) {
            sendMessage();
        }
    });

    // Automatically upload the file once selected
    $('#fileInput').change(function() {
        var file = $('#fileInput')[0].files[0];
        if (file) {
            uploadFile(file); // Trigger file upload immediately
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
            beforeSend: function() {
                // Optionally show loading indicator
            },
            success: function(response) {
                // Optionally update the sidebar with the new file
                updateUploadedFilesList(file.name); // Call function to update file list
                showNotification('Documents uploaded successfully', 'success');
            },
            error: function() {
                console.error('Error uploading file.');
            }
        });
    }

    // Function to update the uploaded files list in the sidebar
    function updateUploadedFilesList(fileName) {
        var uploadedFilesList = $('.sidebar ul');
        uploadedFilesList.append('<li>' + fileName + '</li>');
    }

    $('#sendButton').click(function() {
        sendMessage();
    });

    function sendMessage() {
        var userInput = $('#userInput').val();
        if (userInput) {
            $('#messages').append('<div class="message user-message">' + userInput + '</div>');
            $('#userInput').val('');
            
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
