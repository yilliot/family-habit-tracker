<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('sprint_participants', function (Blueprint $table) {
            $table->id();
            $table->foreignId('sprint_id')->constrained('sprints')->cascadeOnDelete();
            $table->foreignId('person_id')->constrained('people')->restrictOnDelete();
            $table->unsignedInteger('carry_forward_balance')->default(0);
            $table->foreignId('active_reward_id')->nullable()->constrained('rewards')->nullOnDelete();
            $table->unsignedInteger('display_order')->default(0);
            $table->timestamps();

            $table->unique(['sprint_id', 'person_id']);
            $table->index('display_order');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('sprint_participants');
    }
};
