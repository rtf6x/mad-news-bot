const carAdviceProtoResults = [
  'Не бери жука, там 1.2 движок на 1.4 веса, и экологический класс D',
  'YOLO, что хочется - то и бери!',
  'Любишь кататься - люби и катайся!',
  'Какой ответ ожидаешь ты, юный падаван? Выбрать сам способен ибо сила ведёт тебя. Но остерегайся стороны тёмной влияния',
  'PSA - зло. Французы умеют делать только дизель',
  'Вам шашечки, или ехать?',
  'Машина бывает Тойота и стиральная',
  'Сухая DSG - не течёт',
];
let carAdviceResults = [];

export default async function currency() {
  if (!carAdviceResults.length) {
    carAdviceResults = JSON.parse(JSON.stringify(carAdviceProtoResults));
  }
  return carAdviceResults.splice(Math.floor(Math.random() * carAdviceResults.length), 1)[0];
}
