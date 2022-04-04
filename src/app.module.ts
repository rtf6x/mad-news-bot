import { Module } from '@nestjs/common';
import { WebhookService } from './webhook/webhook.service';
import { WebhookController } from './webhook/webhook.controller';

@Module({
  imports: [],
  controllers: [WebhookController],
  providers: [WebhookService],
})
export class AppModule {
}
