document.addEventListener('sign-in-success', function (evt) {
    // 1. Close the modal using your UI store
    if (window.Alpine && Alpine.store('ui')) {
        Alpine.store('ui').modal.open = false;
    }

    // 2. Optional: Show a quick toast or just redirect
    console.log("Signin successful, redirecting...");

    // 3. Perform the redirect
    // We use a small delay so the user sees the 'Success' component for a split second
    setTimeout(() => {
        window.location.href = "/dashboard";
    }, 500);
});

// Also handle the signupSuccess the same way
document.addEventListener('sign-up-success', function (evt) {
    setTimeout(() => {
        window.location.href = "/welcome";
    }, 500);
});