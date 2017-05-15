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

function hide_settings() {
  $("#left-menu").animate(
    {
      width: "0px"
    },
    400,
    function() {
      $("#left-menu").hide();
    }
  );
}

function toggle_settings() {
  if ($("#left-menu").is(":visible")) {
    hide_settings();
  } else {
    $("#left-menu").show();
    $("#left-menu").animate(
      {
        width: "350px"
      },
      400
    );
  }
}

function webui_init() {
  // Palmi
  $("#palmi").submit(function(e) {
    post_msg(e);
    e.preventDefault();
  });

  $("#pini").on(
    {
      click: function(e) {
        norlogeclicked(e);
      },
      mouseenter: function(e) {
        norlogeHighlight(e);
      },
      mouseleave: function(e) {
        clearHighlight(e);
      }
    },
    ".post_clock"
  );

  // Pini
  $("#pini").on(
    {
      mouseenter: function(e) {
        let q = document.querySelectorAll(":hover");
        let totozTxt = q[q.length - 1].innerText.slice(2, -1); // Surrounding "[:" & "]"

        let totozSrv = totozServer ? totozServer : TOTOZ_DEFAULT_SERVER;

        let popup = $("#popup");
        if (popup.is(":visible")) {
          popup.hide();
        }

        let oImg = document.createElement("img");
        oImg.setAttribute("src", totozSrv + "/" + encodeURI(totozTxt));
        oImg.setAttribute("alt", "na");

        let popupElt = popup[0];
        popupElt.style.position = "absolute";
        popupElt.style.left = e.pageX.toString() + "px";
        popupElt.style.top = e.pageY.toString() + "px";
        popup.fadeIn(500);
        while (popupElt.firstChild) {
          popupElt.removeChild(popupElt.firstChild);
        }
        popupElt.appendChild(oImg);
        e.preventDefault();
      },
      mouseleave: function(e) {
        e.preventDefault();
        let popup = $("#popup");
        if (popup.is(":visible")) {
          popup.fadeOut(200);
        }
        e.preventDefault();
      }
    },
    ".totoz"
  );

  $("#pini").on(
    {
      mouseenter: function(e) {
        clockRefHighlight(e);
      },
      mouseleave: function(e) {
        clearHighlight(e);
      }
    },
    ".clock_ref"
  );

  $("#pini").on("mouseenter", ".clock_ref", function(e) {
    return false;
  });

  if (typeof TOTOZ_DEFAULT_SERVER === "string") {
    $("#totozServer")[0].placeholder = TOTOZ_DEFAULT_SERVER;
  }

  // Backend API link
  let swagger_url = window.location.protocol + "//" + window.location.host;
  let swagger_base_path = window.location.pathname.replace(/\/webui\/?.*/, SWAGGER_BASE_PATH + "/");
  let swagger_href =
    swagger_url + swagger_base_path + "?url=" + encodeURIComponent(swagger_url + swagger_base_path + SWAGGER_FILE_NAME);
  $("#backend-api-link").attr("href", swagger_href);
  // Settings menu
  $("#settings").on("click", function(e) {
    toggle_settings(e);
    e.stopPropagation();
    e.preventDefault();
  });

  $("#toggler-btn").on("click", function(e) {
    toggle_settings(e);
    e.preventDefault();
  });

  $(document).on("click", function(e) {
    var container = $("#left-menu");

    if (
      !container.is(e.target) && // if the target of the click isn't the container...
      container.has(e.target).length === 0
    ) {
      // ... nor a descendant of the container
      hide_settings();
    }
  });

  // Login form
  $("#login-form").submit(function(e) {
    login(e);
    e.preventDefault();
  });

  $("#login-dp").on("click", "a", function(e) {
    logout(e);
    e.preventDefault();
  });

  whoami();
  update_pini();
  // Launch pini periodic refresh
  intervalID = setInterval(update_pini, PINI_REFRESH_MS);
}

function login() {
  let loginName = $("#loginInput").val();
  let pass = $("#passwordInput").val();

  $.ajax({
    method: "POST",
    url: BASE_PATH + "/user/login",
    contentType: "application/x-www-form-urlencoded",
    data: {
      login: loginName,
      password: pass
    }
  })
    .done(function(data, textStatus, request) {
      show_success("Login successfull");
      whoami();
    })
    .fail(function(jqXHR, textStatus, errorThrown) {
      show_error("Login failed : " + errorThrown);
    });
}

function logout() {
  $.ajax({
    method: "GET",
    url: BASE_PATH + "/user/logout"
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
    method: "POST",
    url: BASE_PATH + "/post",
    contentType: "application/x-www-form-urlencoded",
    // data to be added to query string:
    data: postData,
    // type of data we are expecting in return:
    // dataType: '',
    timeout: 300,
    context: $("body")
  })
    .done(function(data, textStatus, jqXHR) {
      $("#palmiInput").val("");
      // Stop pini refresh
      clearInterval(intervalID);
      update_pini();
      // Relaunch pini periodic refresh
      intervalID = setInterval(update_pini, PINI_REFRESH_MS);
    })
    .fail(function(jqXHR, textStatus, errorThrown) {
      console.log("Ajax error : " + textStatus);
    });
}

/*  Pini */

function norlogeclicked(e) {
  let id = e.target.id;
  let parts = id.split("-");

  let d = new Date(); //Current date
  let currDate = d.getFullYear() + "_" + ("0" + (d.getMonth() + 1)).slice(-2) + "_" + ("0" + d.getDate()).slice(-2);

  let norloge = "";

  //Date if necessary
  if (currDate != parts[0].slice(-10)) {
    let postDate = parts[0].slice(-10);
    if (currDate.slice(0, 4) != postDate.slice(0, 4)) {
      norloge += parts[0].slice(-10).replace(/_/g, "/") + "#";
    } else {
      norloge += parts[0].slice(-5).replace(/_/g, "/") + "#";
    }
  }

  //Time
  norloge += parts[1].replace(/_/g, ":").replace(/^t/, "");

  //Index if necessary
  if (parts.length >= 2 && parts[1] > 1) {
    switch (parts[1]) {
      case "1":
        norloge += "¹";
        break;
      case "2":
        norloge += "²";
        break;
      case "3":
        norloge += "³";
        break;
      default:
        norloge += "^" + parts[1];
    }
  }

  postClockClicked(norloge);
}

function norlogeHighlight(e) {
  let id = e.target.id;
  let parts = id.split("-");
  let norlogeD = parts[0];
  let norlogeDShort = "d" + norlogeD.slice(-5); // For yearless norloges
  let norlogeDLess = "d"; // For dateless norloges
  let norlogeT = parts[1];

  let query_strs = [];

  if (parts.length >= 3 && parts[2].slice(1) > 1) {
    if (norlogeD.length >= 2) {
      //Date
      query_strs.push(norlogeD + "-" + norlogeT + "-" + parts[2]);
      query_strs.push(norlogeDShort + "-" + norlogeT + "-" + parts[2]);
      query_strs.push(norlogeDLess + "-" + norlogeT + "-" + parts[2]);
    } else {
      query_strs.push(norlogeT + "-" + parts[2]);
    }
  } else {
    if (norlogeD.length >= 2) {
      //Date
      query_strs.push(norlogeD + "-" + norlogeT);
      query_strs.push(norlogeD + "-" + norlogeT + "-" + parts[2]);
      query_strs.push(norlogeDShort + "-" + norlogeT);
      query_strs.push(norlogeDShort + "-" + norlogeT + "-" + parts[2]);
      query_strs.push(norlogeDLess + "-" + norlogeT);
      query_strs.push(norlogeDLess + "-" + norlogeT + "-" + parts[2]);
    } else {
      query_strs.push(norlogeT);
      query_strs.push(norlogeT + "-i1");
    }
  }

  query_strs.forEach(function(e) {
    $("#pini").find("span[id$=" + e + "]").each(function(index) {
      if ($(this).hasClass("clock_ref")) {
        $(this).addClass("highlighted");
      } else {
        $(this).parent().addClass("highlighted");
      }
    });
  });
}

function clockRefHighlight(e) {
  let meta = $("span.norloge_ref_meta", e.target);
  let parts = meta[0].innerText.split("|");
  let norlogeD = parts[0].replace(/:/g, "_");
  let norlogeT = parts[1].replace(/:/g, "_");
  let norlogeI = parts[2];

  let query_str = "";
  if (norlogeD.length > 0) {
    query_str += norlogeD + "-";
  }
  query_str += "t" + norlogeT + (norlogeI ? "-i" + norlogeI : "");

  $("#pini").find("span[id*=" + query_str + "]").each(function(index) {
    if ($(this).hasClass("clock_ref")) {
      $(this).addClass("highlighted");
    } else {
      $(this).parent().addClass("highlighted");
    }
  });
}

function clearHighlight(e) {
  $("#pini").find(".highlighted").each(function(index) {
    $(this).removeClass("highlighted");
  });
  $("#pini").find(".highlighted").each(function(index) {
    $(this).removeClass("highlighted");
  });
}

function update_pini() {
  var url = BACKEND_URL;
  if (url.indexOf("%i")) {
    url = url.replace("%i", maxId.toString());
  }

  $.ajax({
    method: "GET",
    url: url,
    // type of data we are expecting in return:
    dataType: "json",
    timeout: 300,
    context: $("body")
  })
    .done(function(data, textStatus, jqXHR) {
      data.Posts = data.Posts.filter(function(item) {
        return item.id > maxId;
      });

      if (data.Posts.length > 1 && data.Posts[0].id > data.Posts[1].id) {
        data.Posts.sort(function(a, b) {
          return a.id - b.id;
        });
      }

      var pini = $("#pini");
      $.each(data.Posts, function(index, item) {
        maxId = item.id > maxId ? item.id : maxId;

        let d = document.createElement("div");
        d.className = "post";

        let s = document.createElement("span");
        let formatedClock = formatPostClock(new Date(item.time));
        let idClock = "d" + formatedClock.replace(/:/g, "_").replace(/\//g, "_").replace(/#/g, "-t");

        index = 1;
        while ($("#" + idClock + "-i" + index).length > 0) {
          index++;
        }

        s.className = "post_clock";
        s.id = idClock + "-i" + index;
        s.title = item.id;
        s.innerHTML = formatedClock.slice(-8);
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
        msg = emojify(msg);
        s.innerHTML = msg;
        d.appendChild(s);

        pini.append(d);

        //Purge too olds posts
        while (pini.children().length > MAX_POSTS) {
          pini.children(":first-child").remove();
        }
      });

      if (firstLoad) {
        window.scrollTo(0, $("body")[0].scrollHeight);
        firstLoad = false;
      }
    })
    .fail(function(jqXHR, textStatus, errorThrown) {
      console.log("Ajax error!" + errorThrown);
    });
}

function whoami() {
  $.ajax({
    method: "GET",
    url: BASE_PATH + "/user/whoami",
    // type of data we are expecting in return:
    dataType: "json",
    timeout: 300,
    context: $("body")
  })
    .done(function(data, textStatus, jqXHR) {
      var str = "<span>Welcome </span>";
      str += "<strong>" + data.Login + "</strong>";
      str += "<span> </span>";
      str += '<a href="#" class="logout">(logout)</a>';
      $("#login-welcome-auth").html(str);
      $("#login-form-group").hide();
    })
    .fail(function(jqXHR, textStatus, errorThrown) {
      $("#login-welcome-auth").html("<strong>Unauthenticated</strong>");
    });
}

function postClockClicked(clicked_id) {
  let txt = clicked_id + " ";
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
  let Y = date.getFullYear();
  let M = ("0" + (date.getMonth() + 1)).slice(-2);
  let D = ("0" + date.getDate()).slice(-2);
  let h = date.getHours() > 9 ? date.getHours() : "0" + date.getHours();
  let m = date.getMinutes() > 9 ? date.getMinutes() : "0" + date.getMinutes();
  let s = date.getSeconds() > 9 ? date.getSeconds() : "0" + date.getSeconds();
  return Y + "/" + M + "/" + D + "#" + h + ":" + m + ":" + s;
}

function totozify(message) {
  var exp = /\[\:([^\t\)\]]+)\]/g;
  return message.replace(exp, '<span class="totoz">[:$1]</span>');
}

function norlogify(message) {
  let datePart = "(?:[0-9]+/)?(?:1[0-2]|0[1-9])/(?:3[0-1]|[1-2][0-9]|0[1-9])"; // (?:y+/)?(?:(?:mm)/(?:dd));
  let timePart = "(?:2[0-3]|[0-1][0-9]):(?:[0-5][0-9])(?::[0-5][0-9])?"; // (?:hh):(?:mm)(?::ss)?;
  let indexPart = "(?:[¹²³]|[:\\^][1-9]|[:\\^][1-9][0-9])?"; // (?:¹²³|[:^]i|[:^]ii)?;
  let bouchotPart = "(?:@([A-Za-z0-9_]+))";

  let nReg = "(?:(" + datePart + ")#)?" + "(" + timePart + ")" + "(" + indexPart + ")?" + bouchotPart + "?";
  let exp = new RegExp(nReg, "g");

  //Do not expand nhorloges in html links, so we need to tokenize on these tags to
  //  only apply replace outside.
  let aReg = new RegExp("((?:<a)|(?:<\\/a\\s*>))");
  let splits = message.split(aReg);
  let res = "";

  for (let i = 0; i < splits.length; i++) {
    let tmp = splits[i];
    if (i % 4 == 0) {
      tmp = tmp.replace(exp, function(match, date, time, index, dest, offset, string) {
        let d = date ? date.replace(/\//g, "_") : "";
        let t = time.replace(/:/g, "_");
        let i = index
          ? index.replace(/[¹²³^]/, function(m) {
              return {
                "^": "",
                "¹": "1",
                "²": "2",
                "³": "3"
              }[m];
            })
          : "";

        return (
          '<span class="clock_ref" id="d' +
          d +
          "-t" +
          t +
          (i ? "-i" + i : "") +
          '"><span class="norloge_ref_meta">' +
          d +
          "|" +
          t +
          "|" +
          i +
          "|" +
          (dest ? dest : "") +
          "</span>" +
          match +
          "</span>"
        );
      });
    }
    res += tmp;
  }
  return res;
}

function emojify(message) {
  return emojione.unicodeToImage(message);
}
