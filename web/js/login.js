 $("#login-button").click(function(event){
		 event.preventDefault();
	 
	 $('form').fadeOut(500);
	 $('.wrapper').addClass('form-success');
	 $.ajax({
		 url : "/user/login",
		 data : "username=admin&password=admin",
		 success : function(msg) {
		 	var res = JSON.parse(msg);
		 	if (res.code == 200) {
		 		window.location.href= "/";
			} else {
		 		alert(res.message)
			}
		 }
	 });
});