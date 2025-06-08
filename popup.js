document.getElementById('summarizeBtn').addEventListener('click', async () => {
  const button = document.getElementById('summarizeBtn');
  const messageDiv = document.getElementById('message');
  messageDiv.textContent = '';

  try {
    button.textContent = 'Sending...';
    button.disabled = true;

    // Get the user's email from storage
    const { userEmail } = await new Promise(resolve => {
      chrome.storage.sync.get(['userEmail'], resolve);
    });
    if (!userEmail) {
      messageDiv.textContent = 'Please set your email in Settings.';
      button.textContent = 'Summarize Article';
      button.disabled = false;
      // Automatically open the settings window
      if (chrome.runtime.openOptionsPage) {
        chrome.runtime.openOptionsPage();
      } else {
        window.open('settings.html');
      }
      return;
    }

    // Get the active tab
    const [tab] = await chrome.tabs.query({ active: true, currentWindow: true });
    const url = tab.url;

    // Send the link and email to the backend
    const requestData = { 
      url, 
      email: userEmail
    };
    console.log('Sending request data:', requestData);

    const response = await fetch('https://3c78-2600-1700-7c10-91d0-7c99-72b1-b737-205f.ngrok-free.app/submit-link', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(requestData)
    });

    console.log('Response status:', response.status);
    const responseText = await response.text();
    console.log('Response body:', responseText);

    if (!response.ok) throw new Error(`Failed to send link. Status: ${response.status}, Response: ${responseText}`);

    messageDiv.textContent = 'Link queued. You will receive a summary email after processing.';

    // Update queue count
    await updateQueueCount(userEmail);
  } catch (error) {
    messageDiv.textContent = 'Error: ' + error.message;
  } finally {
    button.textContent = 'Summarize Article';
    button.disabled = false;
  }
});

// Add settings link functionality
document.getElementById('settingsLink').addEventListener('click', (e) => {
  e.preventDefault();
  if (chrome.runtime.openOptionsPage) {
    chrome.runtime.openOptionsPage();
  } else {
    window.open('settings.html');
  }
});


async function updateQueueCount(email) {
  const response = await fetch(`https://3c78-2600-1700-7c10-91d0-7c99-72b1-b737-205f.ngrok-free.app/queue-count?email=${encodeURIComponent(email)}`);
  if (response.ok) {
    const data = await response.json();
    const queueCountDiv = document.getElementById('queueCount');
    if (data.count === 5) {
      queueCountDiv.textContent = "Sent!";
    } else {
      queueCountDiv.textContent = `Queue: ${data.count} / 5`;
    }
  }
}

// In the main click handler, after getting userEmail:
updateQueueCount(userEmail);

chrome.runtime.onMessage.addListener((msg) => {
  if (msg === 'auto-summarize') {
    const btn = document.getElementById('summarizeBtn');
    if (btn && !btn.disabled) {
      btn.click();
    }
  }
});
