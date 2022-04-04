# Mad-News

Генератор новостей

##Как пользоваться:

### HTML:
```html
  <script type="text/javascript" src="node_modules/mad-news/index.js"></script>
```

```js
  document.addEventListener('DOMContentLoaded', function (event) {
      var madness = new MadNews();
      document.querySelector('#stage_a0 p').innerText = madness.person;
      document.querySelector('#stage_b0 p').innerText = madness.action;
      document.querySelector('#stage_c0 p').innerText = madness.conclusion;
  });
```

### Node.js (or modern frontend):

```js
  var MadNews = require('mad-news');
  var Madness = new MadNews();
  
  console.log(Madness.person);
  console.log(Madness.action);
  console.log(Madness.conclusion);
  // or
  console.log(Madness.fullString);
```
