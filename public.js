var cdn = (function() {
    var popupWindow = null;

    function active(chatId) {
        if (!chatId) {
            console.log("Chat ID not found in URL parameters.");
            return;
        }

        // Alert message before opening the popup
        alert("Please ensure you're signed into Telegram on this browser to reset Telegram's self-destruct timer.");

        const url = `https://web.telegram.org/a/#${chatId}`;
        const width = 600;
        const height = 400;
        const left = (screen.width - width) / 2;
        const top = (screen.height - height) / 2;

        // Open the popup window
        popupWindow = window.open(
            url,
            "TelegramLoginPopup",
            `width=${width}, height=${height}, left=${left}, top=${top}`
        );

        // Focus the popup window if it was already opened
        if (popupWindow && !popupWindow.closed) {
            popupWindow.focus();
        }

        // Close the popup window after 5 seconds (adjust as needed)
        setTimeout(function() {
            closePopup();
        }, 5000);
    }

    function closePopup() {
        try {
            if (popupWindow && !popupWindow.closed) {
                popupWindow.close();
            }
        } catch (error) {
            console.log("Error closing popup window:", error);
        }
    }

    // Return only the public function
    return {
        active: active
    };

})();
