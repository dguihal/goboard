/* global $, sessionStorage, twemoji, emojiRenderSettings:true, totozServer, maxId:true, firstLoad:true, intervalID:true */
/* exported webuiInit */
import * as settings from './settings.js'

$(document).ready(function() {webuiInit();})

function showSuccess (msg) {
  $('#success-alert').html(msg)
  $('#success-alert').alert()
  $('#success-alert')
    .fadeTo(2000, 500)
    .slideUp(500, function () {
      $('#success-alert').slideUp(500)
    })
}

function showError (msg) {
  $('#danger-alert').html(msg)
  $('#danger-alert').alert()
  $('#danger-alert')
    .fadeTo(2000, 500)
    .slideUp(500, function () {
      $('#danger-alert').slideUp(500)
    })
}

function hideSettings () {
  $('#left-menu').animate(
    {
      width: '0px'
    },
    400,
    function () {
      $('#left-menu').hide()
    }
  )
}

function toggleSettings () {
  if ($('#left-menu').is(':visible')) {
    hideSettings()
  } else {
    $('#left-menu').show()
    $('#left-menu').animate(
      {
        width: '350px'
      },
      400
    )
  }
}

export function webuiInit () {
  // Palmi
  $('#palmi').submit(function (e) {
    postMsg(e)
    e.preventDefault()
  })

  // Pini
  $('#pini').on(
    {
      click: function (e) {
        norlogeclicked(e)
      },
      mouseenter: function (e) {
        norlogeHighlight(e)
      },
      mouseleave: function (e) {
        clearHighlight(e)
      }
    },
    '.post-clock'
  )

  if (sessionStorage.getItem('emojiMode') !== null) {
    const emojiMode = sessionStorage.getItem('emojiMode')
    if (emojiMode === 'png') {
      emojiRenderSettings = {
        base: './images/emojis/',
        ext: '.png'
      }
      $('#emoji-mode').val('png')
    } else if (emojiMode === 'svg') {
      emojiRenderSettings = {
        base: './images/emojis/',
        folder: 'svg',
        ext: '.svg'
      }
      $('#emoji-mode').val('svg')
    }
  }

  $('#emoji-mode').on({
    change: function (e) {
      const emojiMode = $('option:selected', this).attr('value')
      if (emojiMode === 'png') {
        emojiRenderSettings = {
          base: './images/emojis/',
          ext: '.png'
        }
        sessionStorage.setItem('emojiMode', 'png')
      } else if (emojiMode === 'svg') {
        emojiRenderSettings = {
          base: './images/emojis/',
          folder: 'svg',
          ext: '.svg'
        }
        sessionStorage.setItem('emojiMode', 'svg')
      }
      // ToDo : Recompute pini
    }
  })

  $('#pini').on(
    {
      mouseenter: function (e) {
        const srcElt = this
        const totozTxt = srcElt.innerText.slice(2, -1) // Surrounding "[:" & "]"

        const totozSrv = totozServer || settings.TOTOZ_DEFAULT_SERVER

        const popup = $('#popup')
        if (popup.is(':visible')) {
          popup.hide()
        }

        const oImg = document.createElement('img')
        oImg.setAttribute('src', totozSrv + '/' + encodeURI(totozTxt))
        oImg.setAttribute('alt', 'Unknown Totoz')

        const popupElt = popup[0]
        popupElt.style.position = 'absolute'
        popupElt.style.left =
          (srcElt.offsetLeft + srcElt.offsetWidth + 1).toString() + 'px'
        popupElt.style.top = srcElt.offsetTop + 'px'
        popup.fadeIn(500)
        while (popupElt.firstChild) {
          popupElt.removeChild(popupElt.firstChild)
        }
        popupElt.appendChild(oImg)
        e.preventDefault()
      },
      mouseleave: function (e) {
        e.preventDefault()
        const popup = $('#popup')
        if (popup.is(':visible')) {
          popup.fadeOut(200)
        }
        e.preventDefault()
      }
    },
    '.totoz'
  )

  $('#pini').on(
    {
      mouseenter: function (e) {
        clockRefHighlight(e)
      },
      mouseleave: function (e) {
        clearHighlight(e)
      }
    },
    '.clock-ref'
  )

  $('#pini').on('mouseenter', '.clock-ref', function (e) {
    return false
  })

  // Totoz Server
  if (typeof settings.TOTOZ_DEFAULT_SERVER === 'string') {
    $('#totoz-server')[0].placeholder = settings.TOTOZ_DEFAULT_SERVER
  }

  // Backend API link
  const swaggerHref = '../swagger/' + settings.SWAGGER_FILE_NAME
  $.ajax({
    url: swaggerHref,
    type: 'GET',
    statusCode: {
      404: function () {
        $('#backend-api-link').hide()
      }
    }
  })
    .done(function (data, textStatus, request) {
      $('#backend-api-link').attr(
        'href',
        '../swagger/?url=' + settings.SWAGGER_FILE_NAME
      )
    })
    .fail(function (jqXHR, textStatus, errorThrown) {
      $('#backend-api-link').hide()
    })

  // Settings menu
  $('#settings').on('click', function (e) {
    toggleSettings(e)
    e.stopPropagation()
    e.preventDefault()
  })

  $('#toggler-btn').on('click', function (e) {
    toggleSettings(e)
    e.preventDefault()
  })

  $(document).on('click', function (e) {
    const container = $('#left-menu')

    if (
      !container.is(e.target) && // if the target of the click isn't the container...
      container.has(e.target).length === 0
    ) {
      // ... nor a descendant of the container
      hideSettings()
    }
  })

  // Login form
  $('#login-form').submit(function (e) {
    login(e)
    e.preventDefault()
  })

  $('#login-dp').on('click', 'a', function (e) {
    logout(e)
    e.preventDefault()
  })

  whoami()
  updatePini()
  // Launch pini periodic refresh
  intervalID = setInterval(updatePini, settings.PINI_REFRESH_MS)
}

function login () {
  const loginName = $('#login-input').val()
  const pass = $('#password-input').val()

  $.ajax({
    method: 'POST',
    url: '../user/login',
    contentType: 'application/x-www-form-urlencoded',
    data: {
      login: loginName,
      password: pass
    }
  })
    .done(function (data, textStatus, request) {
      showSuccess('Login successfull')
      whoami()
    })
    .fail(function (jqXHR, textStatus, errorThrown) {
      showError('Login failed : ' + errorThrown)
    })
}

function logout () {
  $.ajax({
    method: 'GET',
    url: '../user/logout'
  }).always(function () {
    whoami()
    $('#login-form-group').show()
  })
}

function postMsg () {
  let postData = 'message=' + encodeURIComponent($('#palmi-input').val())
  const customUA = $('#info-input').val()
  if (customUA) {
    postData += '&info=' + encodeURIComponent(customUA)
  }

  $.ajax({
    method: 'POST',
    url: settings.POST_URL,
    contentType: 'application/x-www-form-urlencoded',
    // data to be added to query string:
    data: postData,
    // type of data we are expecting in return:
    // dataType: '',
    timeout: 300,
    context: $('body')
  })
    .done(function (data, textStatus, jqXHR) {
      $('#palmi-input').val('')
      // Stop pini refresh
      clearInterval(intervalID)
      updatePini()
      // Relaunch pini periodic refresh
      intervalID = setInterval(updatePini, settings.PINI_REFRESH_MS)
    })
    .fail(function (jqXHR, textStatus, errorThrown) {
      console.log('Ajax error : ' + textStatus)
    })
}

/*  Pini */

function norlogeclicked (e) {
  const id = e.target.id
  const parts = id.split('-')

  const d = new Date() // Current date
  const currDate =
    d.getFullYear() +
    '_' +
    ('0' + (d.getMonth() + 1)).slice(-2) +
    '_' +
    ('0' + d.getDate()).slice(-2)

  let norloge = ''

  // Date if necessary
  if (currDate !== parts[0].slice(-10)) {
    const postDate = parts[0].slice(-10)
    if (currDate.slice(0, 4) !== postDate.slice(0, 4)) {
      norloge += parts[0].slice(-10).replace(/_/g, '/') + '#'
    } else {
      norloge += parts[0].slice(-5).replace(/_/g, '/') + '#'
    }
  }

  // Time
  norloge += parts[1].replace(/_/g, ':').replace(/^t/, '')

  // Index if necessary
  if (parts.length > 2 && parts[2].match(/i\d+/)) {
    const index = parts[2].slice(1)
    switch (index) {
      case '1':
        if ($('#' + id.replace(/1$/, '2'))[0]) {
          // Only if #2 exists
          norloge += '¹'
        }
        break
      case '2':
        norloge += '²'
        break
      case '3':
        norloge += '³'
        break
      default:
        norloge += '^' + index
    }
  }

  postClockClicked(norloge)
}

function norlogeHighlight (e) {
  const id = e.target.id
  const parts = id.split('-')
  const norlogeD = parts[0]
  const norlogeDShort = 'd' + norlogeD.slice(-5) // For yearless norloges
  const norlogeDLess = 'd' // For dateless norloges
  const norlogeT = parts[1]

  const queryStrs = []

  if (parts.length >= 3 && parts[2].slice(1) > 1) {
    if (norlogeD.length >= 2) {
      // Date
      queryStrs.push(norlogeD + '-' + norlogeT + '-' + parts[2])
      queryStrs.push(norlogeDShort + '-' + norlogeT + '-' + parts[2])
      queryStrs.push(norlogeDLess + '-' + norlogeT + '-' + parts[2])
    } else {
      queryStrs.push(norlogeT + '-' + parts[2])
    }
  } else {
    if (norlogeD.length >= 2) {
      // Date
      queryStrs.push(norlogeD + '-' + norlogeT)
      queryStrs.push(norlogeD + '-' + norlogeT + '-' + parts[2])
      queryStrs.push(norlogeDShort + '-' + norlogeT)
      queryStrs.push(norlogeDShort + '-' + norlogeT + '-' + parts[2])
      queryStrs.push(norlogeDLess + '-' + norlogeT)
      queryStrs.push(norlogeDLess + '-' + norlogeT + '-' + parts[2])
    } else {
      queryStrs.push(norlogeT)
      queryStrs.push(norlogeT + '-i1')
    }
  }

  queryStrs.forEach(function (e) {
    highlight('span[id$=' + e + ']') // https://api.jquery.com/attribute-ends-with-selector/
  })
}

function clockRefHighlight (e) {
  const meta = $('span.norloge-ref-meta', e.target)
  const parts = meta[0].innerText.split('|')
  const norlogeD = parts[0].replace(/:/g, '_')
  const norlogeT = parts[1].replace(/:/g, '_')
  const norlogeI = parts[2]

  let queryStr = ''
  if (norlogeD.length > 0) {
    queryStr += norlogeD + '-'
  }
  queryStr += 'span[id*=t' + norlogeT + (norlogeI ? '-i' + norlogeI : '') + ']' // https://api.jquery.com/attribute-contains-selector/
  highlight(queryStr)
}

function highlight (queryStr) {
  $('#pini')
    .find(queryStr)
    .each(function (index) {
      if ($(this).hasClass('clock-ref')) {
        $(this).addClass('highlighted')
      } else {
        $(this).parent().addClass('highlighted')
      }
    })
}

function clearHighlight (e) {
  $('#pini')
    .find('.highlighted')
    .each(function (index) {
      $(this).removeClass('highlighted')
    })
    /*
  $('#pini')
    .find('.highlighted')
    .each(function (index) {
      $(this).removeClass('highlighted')
    })
    */
}

function updatePini () {
  let url = settings.BACKEND_URL
  if (url.indexOf('%i')) {
    url = url.replace('%i', maxId.toString())
  }

  $.ajax({
    method: 'GET',
    url: url,
    // type of data we are expecting in return:
    dataType: 'json',
    timeout: 300,
    context: $('body')
  })
    .done(function (data, textStatus, jqXHR) {
      // Nothing to do if there is no data
      if (!data || !data.Posts) {
        return
      }

      // Remove already known posts from list
      data.Posts = data.Posts.filter(function (item) {
        return item.id > maxId
      })

      // Sort posts by their ids
      if (data.Posts.length > 1 && data.Posts[0].id > data.Posts[1].id) {
        data.Posts.sort(function (a, b) {
          return a.id - b.id
        })
      }

      // Insert posts into pini
      const pini = $('#pini')
      $.each(data.Posts, function (index, item) {
        maxId = item.id > maxId ? item.id : maxId

        const d = document.createElement('div')
        d.className = 'post'

        let s = document.createElement('span')
        const formatedClock = formatPostClock(new Date(item.time))
        const idClock =
          'd' +
          formatedClock
            .replace(/:/g, '_')
            .replace(/\//g, '_')
            .replace(/#/g, '-t')

        index = 1
        while ($('#' + idClock + '-i' + index).length > 0) {
          index++
        }

        s.className = 'post-clock'
        s.id = idClock + '-i' + index
        s.title = item.id
        s.innerHTML = formatedClock.slice(-8)
        d.appendChild(s)

        s = document.createElement('span')
        s.className = item.login.length > 0 ? 'post-login' : 'post-ua'
        s.title = item.info
        s.innerHTML = item.login.length > 0 ? item.login : item.info
        d.appendChild(s)

        s = document.createElement('span')
        s.className = 'post-message'
        let msg = totozify(item.message)
        msg = norlogify(msg)
        s.innerHTML = msg
        twemoji.parse(s, emojiRenderSettings)
        d.appendChild(s)

        pini.append(d)

        // Purge too olds posts
        while (pini.children().length > settings.MAX_POSTS) {
          pini.children(':first-child').remove()
        }
      })

      if (firstLoad) {
        window.scrollTo(0, $('body')[0].scrollHeight)
        firstLoad = false
      }
    })
    .fail(function (jqXHR, textStatus, errorThrown) {
      console.log('Ajax error!' + errorThrown)
    })
}

function whoami () {
  $.ajax({
    method: 'GET',
    url: '../user/whoami',
    // type of data we are expecting in return:
    dataType: 'json',
    timeout: 300,
    context: $('body')
  })
    .done(function (data, textStatus, jqXHR) {
      let str = '<span>Welcome </span>'
      str += '<strong>' + data.Login + '</strong>'
      str += '<span> </span>'
      str += '<a href="#" class="logout">(logout)</a>'
      $('#login-welcome-auth').html(str)
      $('#login-form-group').hide()
    })
    .fail(function (jqXHR, textStatus, errorThrown) {
      $('#login-welcome-auth').html('<strong>Unauthenticated</strong>')
    })
}

function postClockClicked (clickedId) {
  const txt = clickedId + ' '
  insertPalmi(txt)
}

// By now only appends, should take care of caret position
function insertPalmi (string) {
  const palmiInput = $('#palmi-input')
  const caretPos = palmiInput[0].selectionStart
  const caretPosEnd = palmiInput[0].selectionEnd
  const palmiInputTxt = palmiInput.val()

  if (caretPos === caretPosEnd) {
    palmiInput.val(
      palmiInputTxt.substring(0, caretPos) +
        string +
        palmiInputTxt.substring(caretPos)
    )
  } else {
    palmiInput.val(
      palmiInputTxt.substring(0, caretPos) +
        string +
        palmiInputTxt.substring(caretPosEnd)
    )
  }
  palmiInput[0].setSelectionRange(
    caretPos + string.length,
    caretPos + string.length
  )
  $('#palmi-input').focus()
}

function formatPostClock (date) {
  const Y = date.getFullYear()
  const M = ('0' + (date.getMonth() + 1)).slice(-2)
  const D = ('0' + date.getDate()).slice(-2)
  const h = date.getHours() > 9 ? date.getHours() : '0' + date.getHours()
  const m = date.getMinutes() > 9 ? date.getMinutes() : '0' + date.getMinutes()
  const s = date.getSeconds() > 9 ? date.getSeconds() : '0' + date.getSeconds()
  return Y + '/' + M + '/' + D + '#' + h + ':' + m + ':' + s
}

function totozify (message) {
  const exp = /\[:([^\t)\]]+)\]/g
  return message.replace(exp, '<span class="totoz">[:$1]</span>')
}

function norlogify (message) {
  const datePart = '(?:[0-9]+/)?(?:1[0-2]|0[1-9])/(?:3[0-1]|[1-2][0-9]|0[1-9])' // (?:y+/)?(?:(?:mm)/(?:dd));
  const timePart = '(?:2[0-3]|[0-1][0-9]):(?:[0-5][0-9])(?::[0-5][0-9])?' // (?:hh):(?:mm)(?::ss)?;
  const indexPart = '(?:[¹²³]|[:\\^][1-9]|[:\\^][1-9][0-9])?' // (?:¹²³|[:^]i|[:^]ii)?;
  const bouchotPart = '(?:@([A-Za-z0-9_]+))'

  const nReg =
    '(?:(' +
    datePart +
    ')#)?' +
    '(' +
    timePart +
    ')' +
    '(' +
    indexPart +
    ')?' +
    bouchotPart +
    '?'
  const exp = new RegExp(nReg, 'g')

  // Do not expand nhorloges in html links, so we need to tokenize on these tags to
  //  only apply replace outside.
  const splits = message.split(/((?:<a)|(?:<\/a\s*>))/)
  let res = ''

  for (let i = 0; i < splits.length; i++) {
    let tmp = splits[i]
    if (i % 4 === 0) {
      tmp = tmp.replace(
        exp,
        function (match, date, time, index, dest, offset, string) {
          const d = date ? date.replace(/\//g, '_') : ''
          const t = time.replace(/:/g, '_')
          const i = index
            ? index.replace(/[¹²³^]/, function (m) {
                return {
                  '^': '',
                  '¹': '1',
                  '²': '2',
                  '³': '3'
                }[m]
              })
            : ''

          return (
            '<span class="clock-ref" id="d' +
            d +
            '-t' +
            t +
            (i ? '-i' + i : '') +
            '"><span class="norloge-ref-meta">' +
            d +
            '|' +
            t +
            '|' +
            i +
            '|' +
            (dest || '') +
            '</span>' +
            match +
            '</span>'
          )
        }
      )
    }
    res += tmp
  }
  return res
}
