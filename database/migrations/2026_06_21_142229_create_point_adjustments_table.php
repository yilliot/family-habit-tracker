<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('point_adjustments', function (Blueprint $table) {
            $table->id();
            $table->foreignId('sprint_participant_id')->constrained('sprint_participants')->cascadeOnDelete();
            $table->integer('amount');
            $table->string('reason', 120)->nullable();
            $table->timestamp('created_at')->useCurrent();

            $table->index('sprint_participant_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('point_adjustments');
    }
};
