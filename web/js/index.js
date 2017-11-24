/**
 * Created by yuyi on 2017/11/23.
 */
function logout()
{
    $.get("/user/logout");
    window.location.href="/login.html";
}