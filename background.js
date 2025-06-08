chrome.commands.onCommand.addListener((command) => {
  if (command === "summarize-article") {
    chrome.action.openPopup();
    setTimeout(() => {
      chrome.runtime.sendMessage('auto-summarize');
    }, 500); // Wait for popup to open
  }
}); 