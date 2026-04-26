<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('ticks', function (Blueprint $table) {
            $table->id();
            $table->foreignId('habit_id')->constrained('habits')->cascadeOnDelete();
            $table->date('tick_date');
            $table->timestamp('created_at')->useCurrent();

            $table->unique(['habit_id', 'tick_date']);
            $table->index('tick_date');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('ticks');
    }
};
