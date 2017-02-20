function show_success(msg) {
    $("#success-alert").html(msg);
    $("#success-alert").alert();
    $("#success-alert").fadeTo(2000, 500).slideUp(500, function() {
        $("#success-alert").slideUp(500);
    });
}

function show_error(msg) {
    $("#danger-alert").html(msg);
    $("#danger-alert").alert();
    $("#danger-alert").fadeTo(2000, 500).slideUp(500, function() {
        $("#danger-alert").slideUp(500);
    });
}

function login() {
    var login = $('#loginInput').val();
    var pass = $('#passwordInput').val();

    $.ajax({
        method: 'POST',
        url: '/user/login',
        contentType: 'application/x-www-form-urlencoded',
        data: {
            login: login,
            password: pass
        }
    }).done(function(data, textStatus, request) {
        show_success("Login successfull")
        whoami();
    }).fail(function(jqXHR, textStatus, errorThrown) {
        show_error("Login failed : " + errorThrown)
    })
}

function logout() {
    $.ajax({
        method: 'GET',
        url: '/user/logout'
    }).always(function() {
        whoami();
        $("#login-form-group").show();
    });
}

function post_msg() {
    let postData = "message=" + encodeURIComponent($("#palmiInput").val());
    let customUA = $("#InfoInput").val();
    if (customUA) {
        postData += "&info=" + encodeURIComponent(customUA);
    }

    $.ajax({
        method: 'POST',
        url: '//localhost:8080/post',
        contentType: 'application/x-www-form-urlencoded',
        // data to be added to query string:
        data: postData,
        // type of data we are expecting in return:
        // dataType: '',
        timeout: 300,
        context: $('body')
    }).done(function(data, textStatus, jqXHR) {
        $("#palmiInput").val("");
        // Stop pini refresh
        clearInterval(intervalID);
        update_pini();
        // Relaunch pini periodic refresh
        intervalID = setInterval(update_pini, PINI_REFRESH_MS);
    }).fail(function(jqXHR, textStatus, errorThrown) {
        console.log('Ajax error : ' + textStatus)
    });
}

function update_pini() {
    var url = BACKEND_URL;
    if (url.indexOf("%i")) {
        url = url.replace("%i", maxId.toString());
    }

    $.ajax({
        method: 'GET',
        url: url,
        // type of data we are expecting in return:
        dataType: 'json',
        timeout: 300,
        context: $('body')
    }).done(function(data, textStatus, jqXHR) {
        data.Posts = data.Posts.filter(function(item) {
            return item.id > maxId;
        });

        if (data.Posts.length > 1 &&
            data.Posts[0].id > data.Posts[1].id) {

            data.Posts.sort(function(a, b) {
                return a.id - b.id;
            })
        }

        var pini = $("#pini");
        $.each(data.Posts, function(index, item) {
            maxId = item.id > maxId ? item.id : maxId;

            let d = document.createElement("div");
            d.className = "post";

            let s = document.createElement("span");
            let formatedClock = formatPostClock(new Date(item.time));
            let idClock = 't' + formatedClock.replace(/:/g, "_");

            index = 1;
            while ($("#" + idClock + "-i" + index).length > 0) {
                index++;
            }

            s.className = "post_clock";
            s.id = idClock + '-i' + index;
            s.title = item.id;
            s.innerHTML = formatedClock;
            d.appendChild(s);

            s = document.createElement("span");
            s.className = item.login.length > 0 ? "post_login" : "post_ua";
            s.title = item.info;
            s.innerHTML = item.login.length > 0 ? item.login : item.info;
            d.appendChild(s);

            s = document.createElement("span");
            s.className = "post_message";
            let msg = totozify(item.message);
            msg = norlogify(msg);
            s.innerHTML = msg;
            d.appendChild(s);

            pini.append(d);

            //Purge too olds posts	
            while (pini.children().length > MAX_POSTS) {
                pini.children(':first-child').remove();
            }
        });

        if (firstLoad) {
            window.scrollTo(0, $("body")[0].scrollHeight);
            firstLoad = false;
        }
    }).fail(function(jqXHR, textStatus, errorThrown) {
        console.log('Ajax error!' + errorThrown)
    })
}

function whoami() {
    $.ajax({
        method: 'GET',
        url: '/user/whoami',
        // type of data we are expecting in return:
        dataType: 'json',
        timeout: 300,
        context: $('body')
    }).done(function(data, textStatus, jqXHR) {
        var str = "<span>Welcome </span>";
        str += "<strong>" + data.Login + "</strong>"
        str += "<span> </span>"
        str += "<a href=\"#\" class=\"logout\">(logout)</a>"
        $("#login-welcome-auth").html(str);
        $("#login-form-group").hide();
    }).fail(function(jqXHR, textStatus, errorThrown) {
        $("#login-welcome-auth").html("<strong>Unauthenticated</strong>");
    });
}

function postClockClicked(clicked_id) {
    let txt = clicked_id + ' ';
    insertPalmi(txt);
}

//By now only appends, should take care of caret position
function insertPalmi(string) {
    let palmiInput = $("#palmiInput");
    let caretPos = palmiInput[0].selectionStart;
    let caretPosEnd = palmiInput[0].selectionEnd;
    let palmiInputTxt = palmiInput.val();

    if (caretPos == caretPosEnd) {
        palmiInput.val(palmiInputTxt.substring(0, caretPos) + string + palmiInputTxt.substring(caretPos));
    } else {
        palmiInput.val(palmiInputTxt.substring(0, caretPos) + string + palmiInputTxt.substring(caretPosEnd));
    }
    palmiInput[0].setSelectionRange(caretPos + string.length, caretPos + string.length);
    $("#palmiInput").focus();
}

function formatPostClock(date) {
    let h = date.getHours() > 9 ? date.getHours() : "0" + date.getHours();
    let m = date.getMinutes() > 9 ? date.getMinutes() : "0" + date.getMinutes();
    let s = date.getSeconds() > 9 ? date.getSeconds() : "0" + date.getSeconds();
    return h + ":" + m + ":" + s;
}

function totozify(message) {
    var exp = /\[\:([^\t\)\]]+)\]/g;
    return message.replace(exp, '<span class="totoz">[:$1]</span>');
}

function norlogify(message) {
    let datePart = "(?:[0-9]+/)?(?:1[0-2]|0[1-9])/(?:3[0-1]|[1-2][0-9]|0[1-9])"; // (?:y+/)?(?:(?:mm)/(?:dd))
    let timePart = "(?:2[0-3]|[0-1][0-9]):(?:[0-5][0-9])(?::[0-5][0-9])?"; // (?:hh):(?:mm)(?::ss)?
    let indexPart = "(?:[¹²³]|[:\\^][1-9]|[:\\^][1-9][0-9])?" // (?:¹²³|[:^]i|[:^]ii)?
    let bouchotPart = "(?:@([A-Za-z0-9_]+))"

    let nReg = "(?:(" + datePart + ")#)?" + "(" + timePart + ")" + "(" + indexPart + ")?" + bouchotPart + "?"
    //console.log(nReg);
    let exp = new RegExp(nReg, "g")
    //console.log(exp.test("10:10:10"));
    return message.replace(exp, function(match, date, time, index, dest, offset, string) {
        let d = (date ? date.replace(/\//g, "_") : '');
        let t = time.replace(/:/g, "_");
        let i = (index ? index.replace(/[¹²³^]/, function(m) {
            return {
                '^': '',
                '¹': '1',
                '²': '2',
                '³': '3'
            }[m];
        }) : '');
        return "<span class=\"clock_ref\" id=\"d" + d +
            '-t' + t +
            (i ? '-i' + i : '') +
            "\"><span class=\"norloge_ref_meta\">" +
            d + "|" +
            t + "|" +
            i + "|" +
            (dest ? dest : '') + "</span>" +
            match + "</span>";
    });
}