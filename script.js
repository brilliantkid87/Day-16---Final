function sendEmail() {
    var name = document.getElementById("name").value;
    var email = document.getElementById("email").value;
    var phone = document.getElementById("phone").value;
    var subject = document.getElementById("subject").value;
    var message = document.getElementById("message").value;

    var mailtoLink = "mailto:brilliantkid87@gmail.com" + "?subject=" + encodeURIComponent(subject) + "&body=" + encodeURIComponent("Name: " + name + "\nEmail: " + email + "\nPhone Number: " + phone + "\n\nMessage: \n" + message);

    window.location.href = mailtoLink;
}

