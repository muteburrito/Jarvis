$(document).ready(function() {

    // Set marked.js options
    marked.setOptions({
        breaks: true,
        highlight: function (code, lang) {
            return hljs.highlightAuto(code).value;
        }
    });

    // Load chat history if it exists in localStorage
    if (localStorage.getItem('chatHistory')) {
        $('#messages').html(localStorage.getItem('chatHistory'));
        $('.chat-box').scrollTop($('.chat-box')[0].scrollHeight);
    }

    // Function to clear chat history
    $('#clearChatHistoryButton').click(function() {
        localStorage.removeItem('chatHistory');
        $('#messages').empty();
    });

    // Function to render markdown safely
    function renderMarkdown(markdownText) {
        return marked.parse(markdownText);
    }

    // Save chat history
    function updateChatHistory() {
        var chatHistory = $('#messages').html();
        localStorage.setItem('chatHistory', chatHistory);
    }

    $('#sendButton').click(function() {
        sendMessage();
    });

    // Voice Input Functionality
    $('#voiceButton').click(function() {
        const recognition = new (window.SpeechRecognition || window.webkitSpeechRecognition)();
        recognition.lang = 'en-US';
        recognition.start();

        recognition.onresult = function(event) {
            const voiceQuery = event.results[0][0].transcript;
            $('#userInput').val(voiceQuery);
            sendMessage();
        };

        recognition.onerror = function(event) {
            alert('Voice recognition error: ' + event.error);
        };
    });

    // Function to speak the bot's response
    function speakResponse(text) {
        const speech = new SpeechSynthesisUtterance(text);
        window.speechSynthesis.speak(speech);
    }

    $('#userInput').keypress(function(e) {
        if (e.which == 13) {
            sendMessage();
        }
    });

    // Automatically upload the files once selected
    $('#fileInput').change(function() {
        var files = $('#fileInput')[0].files;
        if (files.length > 0) {
            for (var i = 0; i < files.length; i++) {
                var file = files[i];
                if (file.type === "application/pdf") {
                    uploadFile(file);
                } else {
                    alert("Only PDF files are allowed.");
                }
            }
        }
    });

    startTyping();

    // Set interval for backspace and re-typing every 60 seconds
    setInterval(function() {
        backspaceAndRetype();
    }, 60000);

    function startTyping() {
        $('.title').css('width', '0');
        setTimeout(function() {
            $('.title').addClass('typing');
            setTimeout(function() {
                $('.title').css('border-right', 'none');
            }, 2500);
        }, 200);
    }

    function backspaceAndRetype() {
        $('.title').css('border-right', '4px solid #ff79c6');
        $('.title').css('width', '15ch');
        $('.title').removeClass('typing').css('animation', 'backspace 2s steps(15, end) forwards');
        
        setTimeout(function() {
            $('.title').css('animation', 'typing 1s steps(15, end) forwards').addClass('typing');
            setTimeout(function() {
                $('.title').css('border-right', 'none');
            }, 2500);
        }, 2000);
    }

    function showNotification(message, type) {
        var notificationBar = $('.notification-bar');
        notificationBar.text(message);

        if (type === 'success') {
            notificationBar.removeClass('error').addClass('success');
        } else if (type === 'error') {
            notificationBar.removeClass('success').addClass('error');
        }

        notificationBar.addClass('show');

        setTimeout(function() {
            notificationBar.removeClass('show');
        }, 3000);
    }

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
                    fileList.empty();
                    response.uploaded_files.forEach(function(file) {
                        fileList.append('<li>' + file + '</li>');
                    });
                }
            },
            error: function(error) {
                console.error('Error uploading files:', error);
            }
        });
    }

    fetchUploadedFiles();

    function fetchUploadedFiles() {
        $.ajax({
            url: '/list-files',
            type: 'GET',
            success: function(response) {
                if (response.uploaded_files) {
                    var fileList = $('#fileList');
                    fileList.empty();
                    response.uploaded_files.forEach(function(file) {
                        fileList.append('<li>' + file + '</li>');
                    });
                }
            },
            error: function(error) {
                console.error('Error fetching file list:', error);
            }
        });
    }

    $('#processDataButton').click(function() {
        $('#loading').show();

        $.ajax({
            url: '/process-data',
            type: 'POST',
            success: function(response) {
                $('#loading').hide();
                showNotification('Data processed successfully, vector store created!', 'success');
            },
            error: function() {
                $('#loading').hide();
                showNotification('Error processing data.', 'error');
            }
        });
    });

    $('#clearVectorStoreButton').click(function() {
        $.ajax({
            url: '/clear-vector-store',
            type: 'POST',
            success: function(response) {
                showNotification('Vector store cleared successfully!', 'success');
            },
            error: function() {
                showNotification('Error clearing vector store.', 'error');
            }
        });
    });

    $('#clearFilesButton').click(function() {
        $.ajax({
            url: '/clear-docs',
            type: 'POST',
            success: function(response) {
                $('#fileList').empty();
                showNotification('Documents cleared successfully!', 'success');
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
            
            updateChatHistory();

            $('#sendButton').prop('disabled', true);
            $('#sendButton i').removeClass('fa-paper-plane').addClass('fa-spinner fa-spin');
    
            $('#messages').append('<div class="message bot-message thinking-dots"><span class="thinking-dot">.</span><span class="thinking-dot">.</span><span class="thinking-dot">.</span></div>');
    
            $.ajax({
                url: '/ask',
                type: 'POST',
                contentType: 'application/json',
                data: JSON.stringify({ query: userInput }),
                success: function(response) {
                    $('.thinking-dots').remove();
                    
                    console.log(response.response);

                    var markdownResponse = renderMarkdown(response.response);
                    
                    $('#messages').append('<div class="message bot-message">' + markdownResponse + '<button class="btn btn-secondary speak-btn">🔊</button></div>');
                    
                    updateChatHistory();

                    $('#sendButton').prop('disabled', false);
                    $('#sendButton i').removeClass('fa-spinner fa-spin').addClass('fa-paper-plane');
    
                    $('.chat-box').scrollTop($('.chat-box')[0].scrollHeight);
                },
                error: function() {
                    $('.thinking-dots').remove();
                    $('#messages').append('<div class="message bot-message">Error getting response.</div>');
                    
                    $('#sendButton').prop('disabled', false);
                    $('#sendButton i').removeClass('fa-spinner fa-spin').addClass('fa-paper-plane');
    
                    $('.chat-box').scrollTop($('.chat-box')[0].scrollHeight);
                }
            });
        }
    }

    $(document).on('click', '.speak-btn', function() {
        const responseText = $(this).prev().text();
        speakResponse(responseText);
    });
});
