document.addEventListener('DOMContentLoaded', () => {
  const emailInput = document.getElementById('email');
  const saveBtn = document.getElementById('saveBtn');
  const status = document.getElementById('status');

  // Load saved email
  chrome.storage.sync.get(['userEmail'], (result) => {
    if (result.userEmail) {
      emailInput.value = result.userEmail;
    }
  });

  // Validate email on input
  emailInput.addEventListener('input', () => {
    const isValid = emailInput.checkValidity();
    saveBtn.disabled = !isValid;
    if (!isValid) {
      status.textContent = 'Please enter a valid email address';
      status.className = 'error';
      status.style.display = 'block';
    } else {
      status.style.display = 'none';
    }
  });

  // Function to create a falling duck
  function createDuck() {
    const duck = document.createElement('div');
    duck.className = 'duck';
    duck.textContent = '\uD83E\uDD86';
    // Make ducks fall from anywhere across the full window width
    const windowWidth = window.innerWidth;
    duck.style.left = Math.random() * (windowWidth - 32) + 'px'; // 32px is approx duck width
    duck.style.animationDuration = (Math.random() * 1 + 0.5) + 's'; // Random fall speed
    document.body.appendChild(duck);
    
    // Remove the duck after animation
    duck.addEventListener('animationend', () => {
      duck.remove();
    });
  }

  // Function to start the duck rain
  function startDuckRain() {
    const duckCount = 1000; // Number of ducks
    const interval = 200; // Time between ducks in ms
    
    for (let i = 0; i < duckCount; i++) {
      setTimeout(createDuck, i * interval);
    }
  }

  saveBtn.addEventListener('click', () => {
    const email = emailInput.value.trim();
    if (!email || !emailInput.checkValidity()) {
      status.textContent = 'Please enter a valid email address';
      status.className = 'error';
      status.style.display = 'block';
      return;
    }

    // Disable the button while saving
    saveBtn.disabled = true;
    saveBtn.textContent = 'Saving...';

    chrome.storage.sync.set({ userEmail: email }, () => {
      status.textContent = 'Email saved successfully!';
      status.className = 'success';
      status.style.display = 'block';
      saveBtn.textContent = 'Save';
      saveBtn.disabled = false;

      // Start the duck rain
      startDuckRain();
    });
  });
}); 