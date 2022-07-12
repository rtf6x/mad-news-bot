import { NestFactory } from '@nestjs/core';
import { ExpressAdapter } from '@nestjs/platform-express';
import * as express from 'express';
import * as bodyParser from 'body-parser';
import * as morgan from 'morgan';

import { AppModule } from './app.module';
import settings from './settings';

async function bootstrap() {
  const app = new ExpressAdapter();

  app.set('json spaces', 2);
  app.set('trust proxy', true);
  app.disable('x-powered-by');
  app.use(express.static('client'));
  app.use(morgan('dev'));

  app.use(bodyParser.json());

  const nest = await NestFactory.create(AppModule, new ExpressAdapter(app), { cors: true });
  // nest.useGlobalFilters(new HttpExceptionFilter());

  await nest.listen(settings.port, '127.0.0.1');
}

bootstrap();
