const predict = [
  'navi',
  'skinswipe',
  'hltv',
  'steam',
  'рособоронэкспорт',
  'роскосмос',
  'сколково',
  'nasa',
  'blizzard',
  'известная киберспортивная команда из китая',
  'неизвестная киберспортивная организация из мьянмы',
  'малоизвестная команда кибератлетов из бирюлёво',
];

const action = [
  'собирается',
  'планирует',
  'изучает возможность',
];

const conclusion = [
  'нанять ещё одного программиста',
  'уволить сотрудника',
  'нанять уборщицу',
  'выйти на ipo',
  'купить акции apple',
  'купить акции google',
  'купить акции xiaomi, или хотя бы электросамокат',
  'провести турнир по cs:go в открытом космосе',
  'купить сервер в германии',
  'купить astralis',
  'нанять медведева в качестве талисмана на киберспортивной олимпиаде 2080',
  'ничего не делать в этом году',
  'победить конкурентов через 3 года',
  'выпустить пельмени с именами игроков astralis',
];

export default function MadRumors() {
  this.getPerson = function () {
    if (!this.predict || !this.predict.length) {
      this.predict = JSON.parse(JSON.stringify(predict));
    }
    return this.predict.splice(Math.floor(Math.random() * this.predict.length), 1)[0].toUpperCase();
  };

  this.getAction = function () {
    if (!this.action || !this.action.length) {
      this.action = JSON.parse(JSON.stringify(action));
    }
    return this.action.splice(Math.floor(Math.random() * this.action.length), 1)[0].toUpperCase();
  };

  this.getConclusion = function () {
    if (!this.conclusion || !this.conclusion.length) {
      this.conclusion = JSON.parse(JSON.stringify(conclusion));
    }
    return this.conclusion.splice(Math.floor(Math.random() * this.conclusion.length), 1)[0].toUpperCase();
  };

  return `"${this.getPerson().trim()} ${this.getAction().trim()} ${this.getConclusion().trim()}" - Anonymous`;
}
